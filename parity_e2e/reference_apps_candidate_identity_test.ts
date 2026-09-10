import { expect, test } from "@playwright/test";
import { spawnSync } from "node:child_process";
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import {
  assertCandidateSHA,
  assertReferenceAppIdentityStable,
  assertResolvedCandidateModule,
  CandidateSourceCopyLeaseBook,
  CANDIDATE_REPO_ENV,
  CANDIDATE_SHA_ENV,
  createCandidateSourceCopy,
  loadCandidateIdentity,
  resolveCandidateModuleGraph,
  STUDIO_MODULE_PATH,
  type CandidateIdentity,
  type ReferenceAppIdentitySnapshot,
  withCandidateModuleEnvironment,
} from "./reference_apps_candidate_identity";

const candidateSHA = "a".repeat(40);

function temporaryDirectory(label: string): string {
  return mkdtempSync(path.join(tmpdir(), `gosx-studio-candidate-${label}-`));
}

function candidateIdentity(candidateRepo: string): CandidateIdentity {
  return { candidateRepo: path.resolve(candidateRepo), candidateSHA };
}

test.describe("reference-app candidate identity guard", () => {
  test("keeps shared build and server copies leased until groups are gone", () => {
    const leases = new CandidateSourceCopyLeaseBook();
    const hostKey = "/tmp/reference-app-shared-copy";
    const buildLease = leases.retain(hostKey, 101);
    const serverLease = leases.retain(hostKey, 102);

    expect(leases.activeCount(hostKey)).toBe(2);
    expect(leases.canDispose(hostKey)).toBe(false);
    expect(leases.release(buildLease, "live")).toBe(false);
    expect(leases.release(buildLease, "unknown")).toBe(false);
    expect(leases.activeCount(hostKey)).toBe(2);
    expect(leases.release(buildLease, "gone")).toBe(true);
    expect(leases.release(buildLease, "gone")).toBe(false);
    expect(leases.hasActive(hostKey)).toBe(true);
    expect(leases.release(serverLease, "gone")).toBe(true);
    expect(leases.canDispose(hostKey)).toBe(true);
    expect(leases.hasActive(hostKey)).toBe(false);
  });

  test("accepts a graph whose Module.Replace.Dir is the checked-out candidate", () => {
    const candidateRepo = temporaryDirectory("positive");
    try {
      expect(() => assertResolvedCandidateModule({
        Path: STUDIO_MODULE_PATH,
        Replace: { Path: "/tmp/local-replace", Dir: candidateRepo },
      }, candidateIdentity(candidateRepo))).not.toThrow();
    } finally {
      rmSync(candidateRepo, { force: true, recursive: true });
    }
  });

  test("keeps the captured process identity mode and environment immutable", () => {
    const captured: ReferenceAppIdentitySnapshot = {
      mode: "candidate",
      environmentFingerprint: "candidate\u0000sha\u0000GOWORK=off",
      candidateRepo: "/tmp/candidate",
      candidateSHA,
    };
    expect(() => assertReferenceAppIdentityStable(captured, { ...captured })).not.toThrow();
    expect(() => assertReferenceAppIdentityStable(captured, {
      ...captured,
      mode: "unconfigured",
    })).toThrow(/mode changed/);
    expect(() => assertReferenceAppIdentityStable(captured, {
      ...captured,
      environmentFingerprint: "released\u0000version\u0000origin",
    })).toThrow(/environment drift/);
    expect(() => assertReferenceAppIdentityStable(captured, {
      ...captured,
      candidateSHA: "b".repeat(40),
    })).toThrow(/identity changed for candidateSHA/);
  });

  test("fails closed when the graph has no replacement or points at another checkout", () => {
    const candidateRepo = temporaryDirectory("missing-replace");
    const publishedRepo = temporaryDirectory("published");
    try {
      expect(() => assertResolvedCandidateModule({
        Path: STUDIO_MODULE_PATH,
      }, candidateIdentity(candidateRepo))).toThrow(/Module\.Replace\.Dir/);
      expect(() => assertResolvedCandidateModule({
        Path: STUDIO_MODULE_PATH,
        Replace: { Path: "/tmp/published", Dir: publishedRepo },
      }, candidateIdentity(candidateRepo))).toThrow(/expected checked-out candidate/);
    } finally {
      rmSync(candidateRepo, { force: true, recursive: true });
      rmSync(publishedRepo, { force: true, recursive: true });
    }
  });

  test("rejects a candidate SHA mismatch and missing CI SHA before a host can start", () => {
    const candidateRepo = temporaryDirectory("sha-guard");
    try {
      writeFileSync(path.join(candidateRepo, "go.mod"), `module ${STUDIO_MODULE_PATH}\n\ngo 1.26\n`, "utf8");
      expect(() => assertCandidateSHA(candidateSHA, "b".repeat(40))).toThrow(/candidate SHA mismatch/);
      expect(() => loadCandidateIdentity({
        CI: "true",
        GOSX_STUDIO_CANDIDATE_REPO: candidateRepo,
      })).toThrow(/GOSX_STUDIO_CANDIDATE_SHA is required/);
      expect(() => loadCandidateIdentity({
        [CANDIDATE_SHA_ENV]: candidateSHA,
      })).toThrow(new RegExp(`${CANDIDATE_REPO_ENV}.*required`));
    } finally {
      rmSync(candidateRepo, { force: true, recursive: true });
    }
  });

  test("keeps a failed source-copy removal retryable and owned by its exact path", () => {
    const root = temporaryDirectory("copy-removal-retry");
    const hostRepo = path.join(root, "host");
    const candidateRepo = path.join(root, "candidate");
    mkdirSync(hostRepo, { recursive: true });
    mkdirSync(candidateRepo, { recursive: true });
    let copy: ReturnType<typeof createCandidateSourceCopy> | undefined;
    let removalAttempts = 0;
    try {
      writeFileSync(path.join(hostRepo, "go.mod"), `module example.com/reference-app\n\ngo 1.26\n\nrequire ${STUDIO_MODULE_PATH} v0.0.0\n`, "utf8");
      const createdCopy = createCandidateSourceCopy(hostRepo, candidateIdentity(candidateRepo), {
        allowNonGitFixture: true,
        removeSourceRepo: (sourceRepo) => {
          removalAttempts += 1;
          if (removalAttempts === 1) throw new Error("injected removal failure");
          rmSync(sourceRepo, { force: true, recursive: true });
        },
      });
      copy = createdCopy;

      expect(() => createdCopy.dispose()).toThrow(/injected removal failure/);
      expect(existsSync(createdCopy.sourceRepo)).toBe(true);
      expect(() => createdCopy.dispose()).not.toThrow();
      expect(existsSync(createdCopy.sourceRepo)).toBe(false);
      expect(() => createdCopy.dispose()).not.toThrow();
      expect(removalAttempts).toBe(2);
    } finally {
      copy?.dispose();
      rmSync(root, { force: true, recursive: true });
    }
  });

  test("rejects a quoted modfile override and an escaping source symlink", () => {
    const root = temporaryDirectory("copy-safety");
    const hostRepo = path.join(root, "host");
    const candidateRepo = path.join(root, "candidate");
    mkdirSync(hostRepo, { recursive: true });
    mkdirSync(candidateRepo, { recursive: true });
    try {
      writeFileSync(path.join(hostRepo, "go.mod"), `module example.com/reference-app\n\ngo 1.26\n\nrequire ${STUDIO_MODULE_PATH} v0.0.0\n`, "utf8");
      writeFileSync(path.join(candidateRepo, "go.mod"), `module ${STUDIO_MODULE_PATH}\n\ngo 1.26\n`, "utf8");
      expect(() => withCandidateModuleEnvironment({
        GOFLAGS: JSON.stringify("-modfile=/tmp/other.mod"),
      })).toThrow(/-modfile/);

      symlinkSync("../outside-secret", path.join(hostRepo, "source-link"));
      expect(() => createCandidateSourceCopy(hostRepo, candidateIdentity(candidateRepo))).toThrow(/recursive candidate-copy fallback/);
      expect(() => createCandidateSourceCopy(hostRepo, candidateIdentity(candidateRepo), {
        allowNonGitFixture: true,
      })).toThrow(/refusing symlink/);

      mkdirSync(path.join(hostRepo, ".git"), { recursive: true });
      expect(() => createCandidateSourceCopy(hostRepo, candidateIdentity(candidateRepo), {
        allowNonGitFixture: true,
      })).toThrow(/git ls-files failed for Git checkout/);
    } finally {
      rmSync(root, { force: true, recursive: true });
    }
  });

  test("uses independent sibling source copies, preserves relative replaces, and leaves the host untouched", () => {
    const root = temporaryDirectory("copy-root");
    const hostRepo = path.join(root, "reference app space");
    const sharedRepo = path.join(root, "shared-module");
    const candidateRepo = path.join(root, "candidate space");
    mkdirSync(hostRepo, { recursive: true });
    mkdirSync(sharedRepo, { recursive: true });
    mkdirSync(candidateRepo, { recursive: true });
    let firstCopy: ReturnType<typeof createCandidateSourceCopy> | undefined;
    let secondCopy: ReturnType<typeof createCandidateSourceCopy> | undefined;
    try {
      const hostGoMod = `module example.com/reference-app

go 1.26

require (
	m31labs.dev/gosx-studio v0.0.0
	example.com/shared v0.0.0
)

replace example.com/shared => ../shared-module
`;
      const hostGoSum = "example.com/shared v0.0.0 h1:fixture\n";
      writeFileSync(path.join(hostRepo, "go.mod"), hostGoMod, "utf8");
      writeFileSync(path.join(hostRepo, "go.sum"), hostGoSum, "utf8");
      writeFileSync(path.join(hostRepo, "source.txt"), "working-tree source\n", "utf8");
      writeFileSync(path.join(hostRepo, "app.gsx"), "package app\n", "utf8");
      writeFileSync(path.join(hostRepo, ".env"), "SECRET=must-not-copy\n", "utf8");
      writeFileSync(path.join(hostRepo, ".env.example"), "PORT=8080\n", "utf8");
      mkdirSync(path.join(hostRepo, "data"), { recursive: true });
      writeFileSync(path.join(hostRepo, "data", "cms.json"), "runtime data\n", "utf8");
      mkdirSync(path.join(hostRepo, "dist"), { recursive: true });
      writeFileSync(path.join(hostRepo, "dist", "stale.txt"), "generated output\n", "utf8");
      mkdirSync(path.join(hostRepo, ".gosx", "cache"), { recursive: true });
      writeFileSync(path.join(hostRepo, ".gosx", "cache", "stale"), "cache\n", "utf8");
      writeFileSync(path.join(sharedRepo, "go.mod"), "module example.com/shared\n\ngo 1.26\n", "utf8");
      writeFileSync(path.join(candidateRepo, "go.mod"), `module ${STUDIO_MODULE_PATH}\n\ngo 1.26\n`, "utf8");

      const originalGoMod = readFileSync(path.join(hostRepo, "go.mod"));
      const originalGoSum = readFileSync(path.join(hostRepo, "go.sum"));
      const identity = candidateIdentity(candidateRepo);
      firstCopy = createCandidateSourceCopy(hostRepo, identity, { allowNonGitFixture: true });
      secondCopy = createCandidateSourceCopy(hostRepo, identity, { allowNonGitFixture: true });

      expect(firstCopy.sourceRepo).not.toBe(secondCopy.sourceRepo);
      expect(readFileSync(path.join(firstCopy.sourceRepo, "go.mod"), "utf8")).toContain(
        `replace ${STUDIO_MODULE_PATH} => ${JSON.stringify(identity.candidateRepo)}`,
      );
      expect(readFileSync(path.join(firstCopy.sourceRepo, "source.txt"), "utf8")).toBe("working-tree source\n");
      expect(existsSync(path.join(firstCopy.sourceRepo, ".env"))).toBe(false);
      expect(existsSync(path.join(firstCopy.sourceRepo, "data"))).toBe(false);
      expect(existsSync(path.join(firstCopy.sourceRepo, "dist"))).toBe(false);
      expect(existsSync(path.join(firstCopy.sourceRepo, ".gosx"))).toBe(false);
      expect(existsSync(path.join(firstCopy.sourceRepo, ".env.example"))).toBe(true);

      const firstGraph = resolveCandidateModuleGraph(firstCopy.sourceRepo, {
        ...process.env,
        PORT: "first",
        TEMP_DIR: "/tmp/first",
      });
      const secondGraph = resolveCandidateModuleGraph(secondCopy.sourceRepo, {
        ...process.env,
        PORT: "second",
        TEMP_DIR: "/tmp/second",
      });
      expect(() => assertResolvedCandidateModule(firstGraph, identity)).not.toThrow();
      expect(() => assertResolvedCandidateModule(secondGraph, identity)).not.toThrow();

      const firstEnv = withCandidateModuleEnvironment({ ...process.env, PORT: "first", TEMP_DIR: "/tmp/first" });
      const secondEnv = withCandidateModuleEnvironment({ ...process.env, PORT: "second", TEMP_DIR: "/tmp/second" });
      expect(firstEnv.PORT).toBe("first");
      expect(secondEnv.PORT).toBe("second");
      expect(firstEnv.TEMP_DIR).toBe("/tmp/first");
      expect(secondEnv.TEMP_DIR).toBe("/tmp/second");
      expect(firstEnv.GOWORK).toBe("off");

      writeFileSync(path.join(firstCopy.sourceRepo, "source.txt"), "first-only edit\n", "utf8");
      expect(readFileSync(path.join(secondCopy.sourceRepo, "source.txt"), "utf8")).toBe("working-tree source\n");
      firstCopy.dispose();
      expect(existsSync(firstCopy.sourceRepo)).toBe(false);
      expect(existsSync(secondCopy.sourceRepo)).toBe(true);
      expect(readFileSync(path.join(hostRepo, "go.mod"))).toEqual(originalGoMod);
      expect(readFileSync(path.join(hostRepo, "go.sum"))).toEqual(originalGoSum);
    } finally {
      firstCopy?.dispose();
      secondCopy?.dispose();
      rmSync(root, { force: true, recursive: true });
    }
  });

  test("guards every public start helper before build or data allocation", async () => {
    if (process.env.GOSX_STUDIO_PUBLIC_GUARD_PROBE !== "1") {
      const cliPath = require.resolve("@playwright/test/cli");
      const probe = spawnSync(
        process.execPath,
        [cliPath, "test", path.basename(__filename), "--workers=1", "--grep", "guards every public start helper"],
        {
          cwd: __dirname,
          env: {
            ...process.env,
            GOSX_STUDIO_PUBLIC_GUARD_PROBE: "1",
            [CANDIDATE_REPO_ENV]: "",
            [CANDIDATE_SHA_ENV]: "",
            GOSX_STUDIO_RELEASED_CONSUMER: "",
            GOSX_STUDIO_RELEASED_VERSION: "",
            GOSX_STUDIO_RELEASED_ORIGIN: "",
            GOFLAGS: "",
            GOWORK: "",
          },
          encoding: "utf8",
          stdio: ["ignore", "pipe", "pipe"],
        },
      );
      expect(probe.error, `public guard probe failed to spawn: ${probe.stderr}`).toBeUndefined();
      expect(probe.status, `public guard probe output:\n${probe.stdout}\n${probe.stderr}`).toBe(0);
      return;
    }

    const root = temporaryDirectory("public-guard-probe");
    const hostRepo = path.join(root, "synthetic-host");
    const failingGosx = path.join(root, "failing-gosx.sh");
    mkdirSync(hostRepo, { recursive: true });
    writeFileSync(failingGosx, "#!/bin/sh\nprintf 'synthetic guard stub\n' >&2\nexit 73\n", "utf8");
    chmodSync(failingGosx, 0o755);
    const keys = [
      "GOSX_STUDIO_MUDDY_REPO",
      "GOSX_STUDIO_GOSX_BIN",
      CANDIDATE_REPO_ENV,
      CANDIDATE_SHA_ENV,
      "GOSX_STUDIO_RELEASED_CONSUMER",
      "GOSX_STUDIO_RELEASED_VERSION",
      "GOSX_STUDIO_RELEASED_ORIGIN",
      "GOFLAGS",
      "GOWORK",
    ];
    const previous = new Map(keys.map((key) => [key, process.env[key]]));
    try {
      process.env.GOSX_STUDIO_MUDDY_REPO = hostRepo;
      process.env.GOSX_STUDIO_GOSX_BIN = failingGosx;
      for (const key of keys.filter((key) => key !== "GOSX_STUDIO_MUDDY_REPO" && key !== "GOSX_STUDIO_GOSX_BIN")) {
        delete process.env[key];
      }
      // Playwright executes this CommonJS test through its TypeScript loader;
      // require after env setup gives the harness a fresh module capture in
      // this child process without asking Node to parse the .ts file as ESM.
      const harness = require("./reference_apps_harness") as {
        startMuddy: (request: never, extraEnv?: Record<string, string>) => Promise<unknown>;
        startMuddyCollaboration: (request: never, extraEnv?: Record<string, string>) => Promise<unknown>;
        startPajaritos: (request: never) => Promise<unknown>;
      };

      await expect(harness.startMuddy({} as never, { GOWORK: "on" })).rejects.toThrow(/cannot be overridden/);
      await expect(harness.startMuddyCollaboration({} as never, {
        [CANDIDATE_REPO_ENV]: path.join(root, "not-allowed"),
      })).rejects.toThrow(/cannot be overridden/);

      process.env[CANDIDATE_SHA_ENV] = candidateSHA;
      await expect(harness.startPajaritos({} as never)).rejects.toThrow(new RegExp(`${CANDIDATE_REPO_ENV}.*required`));
      delete process.env[CANDIDATE_SHA_ENV];

      // This one deliberate synthetic child proves a normal unconfigured
      // start reaches the configured command only after the guards pass.
      await expect(harness.startMuddy({} as never)).rejects.toThrow(/exit code 73|failed to start/);
      process.env.GOWORK = "on";
      await expect(harness.startMuddy({} as never)).rejects.toThrow(/environment drift/);
    } finally {
      for (const [key, value] of previous) {
        if (value === undefined) delete process.env[key];
        else process.env[key] = value;
      }
      rmSync(root, { force: true, recursive: true });
    }
  });
});
