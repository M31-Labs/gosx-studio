import { expect, test } from "@playwright/test";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import {
  assertCandidateSHA,
  assertResolvedCandidateModule,
  createCandidateSourceCopy,
  loadCandidateIdentity,
  resolveCandidateModuleGraph,
  STUDIO_MODULE_PATH,
  type CandidateIdentity,
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
    } finally {
      rmSync(candidateRepo, { force: true, recursive: true });
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
      expect(() => createCandidateSourceCopy(hostRepo, candidateIdentity(candidateRepo))).toThrow(/refusing symlink/);
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
      firstCopy = createCandidateSourceCopy(hostRepo, identity);
      secondCopy = createCandidateSourceCopy(hostRepo, identity);

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
});
