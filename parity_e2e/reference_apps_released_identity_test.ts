import { expect, test } from "@playwright/test";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import {
  assertReleasedGoModNoReplacement,
  assertReleasedRepositoryModule,
  assertResolvedReleasedModule,
  loadReleasedIdentity,
  RELEASED_CONSUMER_ENV,
  RELEASED_ORIGIN_ENV,
  RELEASED_VERSION_ENV,
  withReleasedModuleEnvironment,
  type ReleasedIdentity,
} from "./reference_apps_released_identity";
import { STUDIO_MODULE_PATH } from "./reference_apps_candidate_identity";

const releasedVersion = "v0.6.2-0.20260828060955-cb313609fb1d";
const releasedOrigin = "cb313609fb1d28937a110dab97698491a05e5030";

function identity(): ReleasedIdentity {
  return { version: releasedVersion, origin: releasedOrigin };
}

test.describe("reference-app released identity guard", () => {
  test("accepts the committed Noni release contract and exact module graph", () => {
    const contract = JSON.parse(
      readFileSync(path.join(__dirname, "../.github/reference-app-release-contract.json"), "utf8"),
    ) as {
      studio: { module: string; version: string; origin: string };
      host: { repository: string; ref: string; sha: string };
    };
    expect(contract.studio.module).toBe(STUDIO_MODULE_PATH);
    expect(contract.studio.version).toBe(releasedVersion);
    expect(contract.studio.origin).toBe(releasedOrigin);
    expect(contract.host.repository).toBe("odvcencio/muddy-noni-commerce");
    expect(contract.host.ref).toBe(contract.host.sha);
    expect(contract.host.sha).toMatch(/^[0-9a-f]{40}$/);

    expect(loadReleasedIdentity({
      [RELEASED_CONSUMER_ENV]: "1",
      [RELEASED_VERSION_ENV]: releasedVersion,
      [RELEASED_ORIGIN_ENV]: releasedOrigin,
    })).toEqual(identity());
    expect(() => assertResolvedReleasedModule({
      Path: STUDIO_MODULE_PATH,
      Version: releasedVersion,
      Origin: { VCS: "git", Hash: releasedOrigin },
    }, identity())).not.toThrow();
  });

  test("fails closed on missing or mismatched published identity", () => {
    expect(loadReleasedIdentity({})).toBeNull();
    expect(() => loadReleasedIdentity({
      [RELEASED_CONSUMER_ENV]: "1",
      [RELEASED_VERSION_ENV]: releasedVersion,
    })).toThrow(/GOSX_STUDIO_RELEASED_ORIGIN is required/);
    expect(() => assertResolvedReleasedModule({
      Path: STUDIO_MODULE_PATH,
      Version: releasedVersion,
      Origin: { Hash: "b".repeat(40) },
    }, identity())).toThrow(/origin mismatch/);
    expect(() => assertResolvedReleasedModule({
      Path: STUDIO_MODULE_PATH,
      Version: "v9.9.9",
      Origin: { Hash: releasedOrigin },
    }, identity())).toThrow(/version mismatch/);
    expect(() => assertResolvedReleasedModule({
      Path: STUDIO_MODULE_PATH,
      Version: releasedVersion,
    }, identity())).toThrow(/Origin\.Hash/);
  });

  test("rejects candidate overlays, module replacements, and GOFLAGS bypasses", () => {
    expect(() => loadReleasedIdentity({
      [RELEASED_CONSUMER_ENV]: "1",
      [RELEASED_VERSION_ENV]: releasedVersion,
      [RELEASED_ORIGIN_ENV]: releasedOrigin,
      GOSX_STUDIO_CANDIDATE_REPO: "/tmp/candidate",
    })).toThrow(/candidate overlay/);
    expect(() => withReleasedModuleEnvironment({
      GOFLAGS: JSON.stringify("-modfile=/tmp/other.mod"),
    })).toThrow(/empty GOFLAGS/);
    expect(() => withReleasedModuleEnvironment({
      GOFLAGS: "-mod=mod",
    })).toThrow(/empty GOFLAGS/);
    expect(withReleasedModuleEnvironment({ PORT: "3811" })).toMatchObject({
      PORT: "3811",
      GOWORK: "off",
      GOFLAGS: "",
    });

    expect(() => assertReleasedGoModNoReplacement(`module example.com/host\n\nrequire ${STUDIO_MODULE_PATH} v0.0.0\n`)).not.toThrow();
    expect(() => assertReleasedGoModNoReplacement(`module example.com/host\n\nreplace ${STUDIO_MODULE_PATH} => ../studio\n`)).toThrow(/replacement/);
    expect(() => assertReleasedGoModNoReplacement(`module example.com/host\n\nreplace (\n\t${STUDIO_MODULE_PATH} v0.0.0 => ../studio\n)\n`)).toThrow(/replacement/);
  });

  test("rejects a replacement field even when version and origin match", () => {
    expect(() => assertResolvedReleasedModule({
      Path: STUDIO_MODULE_PATH,
      Version: releasedVersion,
      Origin: { Hash: releasedOrigin },
      Replace: { Path: STUDIO_MODULE_PATH, Dir: "/tmp/studio" },
    }, identity())).toThrow(/Module\.Replace/);
  });

  test("rejects a direct go.mod version drift before graph resolution", () => {
    const root = mkdtempSync(path.join(tmpdir(), "gosx-studio-released-direct-pin-"));
    try {
      writeFileSync(
        path.join(root, "go.mod"),
        `module example.com/released-host\n\ngo 1.26\n\nrequire ${STUDIO_MODULE_PATH} v0.0.0\n`,
        "utf8",
      );
      expect(() => assertReleasedRepositoryModule(
        root,
        identity(),
        { ...process.env, GOWORK: "off", GOFLAGS: "" },
        "direct pin regression",
      )).toThrow(/direct m31labs\.dev\/gosx-studio require resolved to v0\.0\.0; expected/);
    } finally {
      rmSync(root, { force: true, recursive: true });
    }
  });

  test("rejects a Studio require marked // indirect before graph resolution", () => {
    const root = mkdtempSync(path.join(tmpdir(), "gosx-studio-released-indirect-pin-"));
    try {
      writeFileSync(
        path.join(root, "go.mod"),
        `module example.com/released-host\n\ngo 1.26\n\nrequire ${STUDIO_MODULE_PATH} ${releasedVersion} // indirect\n`,
        "utf8",
      );
      expect(() => assertReleasedRepositoryModule(
        root,
        identity(),
        { ...process.env, GOWORK: "off", GOFLAGS: "" },
        "indirect pin regression",
      )).toThrow(/require is marked \/\/ indirect/);
    } finally {
      rmSync(root, { force: true, recursive: true });
    }
  });
});
