import { execFileSync } from "node:child_process";
import { existsSync, realpathSync, statSync } from "node:fs";
import path from "node:path";
import {
  CANDIDATE_REPO_ENV,
  CANDIDATE_SHA_ENV,
  STUDIO_MODULE_PATH,
  type GoModuleGraph,
} from "./reference_apps_candidate_identity";

/** Explicit opt-in for the released-module/no-overlay reference-app lane. */
export const RELEASED_CONSUMER_ENV = "GOSX_STUDIO_RELEASED_CONSUMER";
export const RELEASED_VERSION_ENV = "GOSX_STUDIO_RELEASED_VERSION";
export const RELEASED_ORIGIN_ENV = "GOSX_STUDIO_RELEASED_ORIGIN";

export type ReleasedIdentity = {
  version: string;
  origin: string;
};

export type ReleasedModuleGraph = GoModuleGraph & {
  Version?: string;
  Origin?: {
    VCS?: string;
    URL?: string;
    Ref?: string;
    Hash?: string;
  };
};

/**
 * Read the explicit published identity used by the released-consumer lane.
 *
 * An unset opt-in is deliberately a no-op so normal local and candidate runs
 * retain their existing behavior. Once enabled, every expected identity
 * value is mandatory and candidate variables are rejected: a released lane
 * must not silently become a PR-overlay run.
 */
export function loadReleasedIdentity(env: NodeJS.ProcessEnv = process.env): ReleasedIdentity | null {
  const enabled = env[RELEASED_CONSUMER_ENV]?.trim();
  if (!enabled) return null;
  if (!/^(?:1|true)$/i.test(enabled)) {
    throw new Error(`${RELEASED_CONSUMER_ENV} must be 1 or true when set`);
  }

  if (env[CANDIDATE_REPO_ENV]?.trim() || env[CANDIDATE_SHA_ENV]?.trim()) {
    throw new Error(
      `${RELEASED_CONSUMER_ENV} cannot run with ${CANDIDATE_REPO_ENV}/${CANDIDATE_SHA_ENV}; refusing a candidate overlay`,
    );
  }

  const version = requiredValue(env[RELEASED_VERSION_ENV], RELEASED_VERSION_ENV);
  if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z][0-9A-Za-z.+-]*)?$/.test(version)) {
    throw new Error(`${RELEASED_VERSION_ENV} must be an exact Go module version; found ${version}`);
  }

  const origin = requiredValue(env[RELEASED_ORIGIN_ENV], RELEASED_ORIGIN_ENV);
  if (!/^[0-9a-f]{40}$/i.test(origin)) {
    throw new Error(`${RELEASED_ORIGIN_ENV} must be a full 40-character git SHA; found ${origin}`);
  }

  return { version, origin: origin.toLowerCase() };
}

/**
 * Assert the exact graph shape required for a published/no-overlay run.
 * `Origin.Hash` is the immutable source identity recorded by Go's module
 * loader; a Replace field would prove that the host is not consuming the
 * released module.
 */
export function assertResolvedReleasedModule(
  module: ReleasedModuleGraph,
  identity: ReleasedIdentity,
): void {
  if (module.Path !== STUDIO_MODULE_PATH) {
    throw new Error(
      `go module graph selected ${module.Path || "<missing module path>"}; expected ${STUDIO_MODULE_PATH}`,
    );
  }
  if (module.Version !== identity.version) {
    throw new Error(
      `released Studio version mismatch: graph resolved ${module.Version || "<missing version>"}; expected ${identity.version}`,
    );
  }

  const originHash = module.Origin?.Hash?.toLowerCase();
  if (!originHash) {
    throw new Error(
      `go module graph did not expose Origin.Hash for ${STUDIO_MODULE_PATH}; refusing an unverifiable published module`,
    );
  }
  if (originHash !== identity.origin) {
    throw new Error(
      `released Studio origin mismatch: graph resolved ${module.Origin?.Hash}; expected ${identity.origin}`,
    );
  }
  if (module.Replace !== undefined && module.Replace !== null) {
    throw new Error(
      `go module graph contains Module.Replace for ${STUDIO_MODULE_PATH}; refusing a local overlay in released mode`,
    );
  }
}

/** Build a fresh environment while rejecting the two known overlay bypasses. */
export function withReleasedModuleEnvironment(baseEnv: NodeJS.ProcessEnv): NodeJS.ProcessEnv {
  if (baseEnv[CANDIDATE_REPO_ENV]?.trim() || baseEnv[CANDIDATE_SHA_ENV]?.trim()) {
    throw new Error(
      `${RELEASED_CONSUMER_ENV} cannot run with candidate environment variables; refusing a candidate overlay`,
    );
  }
  const existingGoFlags = baseEnv.GOFLAGS?.trim() ?? "";
  if (existingGoFlags) {
    const detail = hasModfileFlag(existingGoFlags) ? " (-modfile is forbidden)" : "";
    throw new Error(`released mode requires empty GOFLAGS; found ${existingGoFlags}${detail}`);
  }
  return {
    ...baseEnv,
    GOWORK: "off",
    GOFLAGS: "",
  };
}

/** Reject a Studio local replacement in the host module file itself. */
export function assertReleasedGoModNoReplacement(source: string, label = "host go.mod"): void {
  if (hasStudioReplacement(source)) {
    throw new Error(
      `${label} declares a ${STUDIO_MODULE_PATH} replacement; released mode requires the published module without an overlay`,
    );
  }
}

/** Resolve the module graph from the exact host checkout passed to Go. */
export function resolveReleasedModuleGraph(
  sourceRepo: string,
  baseEnv: NodeJS.ProcessEnv = process.env,
): ReleasedModuleGraph {
  const sourceRoot = resolveDirectory(sourceRepo, "released reference-app repository");
  const commandEnv = withReleasedModuleEnvironment(baseEnv);
  const result = execFileSync(
    "go",
    ["list", "-mod=readonly", "-m", "-json", STUDIO_MODULE_PATH],
    {
      cwd: sourceRoot,
      env: commandEnv,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    },
  );
  try {
    const graph = JSON.parse(result) as ReleasedModuleGraph;
    // `go list -m -json` omits Origin for modules served from a proxy in
    // several Go versions. `go mod download -json` exposes the VCS origin for
    // that same exact version, so fill only the missing metadata and keep the
    // graph itself as the source of Path/Version/Replace truth.
    if (!graph.Origin?.Hash && graph.Version) {
      const downloaded = execFileSync(
        "go",
        ["mod", "download", "-json", `${STUDIO_MODULE_PATH}@${graph.Version}`],
        {
          cwd: sourceRoot,
          env: commandEnv,
          encoding: "utf8",
          stdio: ["ignore", "pipe", "pipe"],
        },
      );
      const download = JSON.parse(downloaded) as { Origin?: ReleasedModuleGraph["Origin"] };
      if (download.Origin) graph.Origin = download.Origin;
    }
    return graph;
  } catch (error) {
    throw new Error(`go list returned invalid released module graph JSON: ${String(error)}`);
  }
}

/**
 * Prove the module file and actual Go graph for one host checkout. This is
 * intentionally callable before and after a build/run so a tool cannot
 * switch from the released module to a replacement during the lifecycle.
 */
export function assertReleasedRepositoryModule(
  sourceRepo: string,
  identity: ReleasedIdentity,
  baseEnv: NodeJS.ProcessEnv = process.env,
  phase = "module graph assertion",
): ReleasedModuleGraph {
  const sourceRoot = resolveDirectory(sourceRepo, "released reference-app repository");
  const modfile = path.join(sourceRoot, "go.mod");
  if (!existsSync(modfile)) {
    throw new Error(`${phase}: released reference-app repository has no go.mod: ${sourceRoot}`);
  }
  assertReleasedGoModStructure(sourceRoot, identity.version, baseEnv, phase);
  const graph = resolveReleasedModuleGraph(sourceRoot, baseEnv);
  assertResolvedReleasedModule(graph, identity);
  return graph;
}

type GoModEditJSON = {
  Require?: Array<{ Path?: string; Version?: string; Indirect?: boolean }>;
  Replace?: Array<{ Old?: { Path?: string } }>;
};

/**
 * Ask Go to parse go.mod rather than relying only on text matching. This
 * catches block/versioned replacement forms while proving Studio is a direct
 * host requirement before the module graph is inspected.
 */
function assertReleasedGoModStructure(
  sourceRoot: string,
  expectedVersion: string,
  baseEnv: NodeJS.ProcessEnv,
  phase: string,
): void {
  let parsed: GoModEditJSON;
  try {
    const result = execFileSync("go", ["mod", "edit", "-json", "go.mod"], {
      cwd: sourceRoot,
      env: withReleasedModuleEnvironment(baseEnv),
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    });
    parsed = JSON.parse(result) as GoModEditJSON;
  } catch (error) {
    throw new Error(`${phase}: Go could not parse the released host go.mod: ${String(error)}`);
  }

  const studioRequirement = parsed.Require?.find((requirement) => requirement.Path === STUDIO_MODULE_PATH);
  if (!studioRequirement) {
    throw new Error(`${phase}: go.mod has no direct require for ${STUDIO_MODULE_PATH}`);
  }
  if (studioRequirement.Indirect === true) {
    throw new Error(
      `${phase}: ${STUDIO_MODULE_PATH} require is marked // indirect; released mode requires an exact direct pin`,
    );
  }
  if (studioRequirement.Version !== expectedVersion) {
    throw new Error(
      `${phase}: direct ${STUDIO_MODULE_PATH} require resolved to ${studioRequirement.Version || "<missing version>"}; expected ${expectedVersion}`,
    );
  }
  if (parsed.Replace?.some((replacement) => replacement.Old?.Path === STUDIO_MODULE_PATH)) {
    throw new Error(
      `${phase}: Go parsed a ${STUDIO_MODULE_PATH} replacement; released mode requires the published module without an overlay`,
    );
  }
}

function requiredValue(value: string | undefined, label: string): string {
  const trimmed = value?.trim();
  if (!trimmed) throw new Error(`${label} is required when ${RELEASED_CONSUMER_ENV}=1`);
  return trimmed;
}

function resolveDirectory(value: string, label: string): string {
  let resolved: string;
  try {
    resolved = realpathSync(path.resolve(value));
  } catch (error) {
    throw new Error(`${label} does not resolve to a directory: ${value} (${String(error)})`);
  }
  try {
    if (!statSync(resolved).isDirectory()) throw new Error("path is not a directory");
  } catch (error) {
    throw new Error(`${label} does not resolve to a directory: ${value} (${String(error)})`);
  }
  return resolved;
}

function hasStudioReplacement(source: string): boolean {
  const escapedModulePath = STUDIO_MODULE_PATH.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(
    `(?:^|\\n)\\s*(?:${escapedModulePath}(?:\\s+v[^\\s]+)?\\s*=>|replace\\s+${escapedModulePath}(?:\\s+v[^\\s]+)?\\s*=>)`,
    "m",
  ).test(source);
}

function hasModfileFlag(flags: string): boolean {
  // Includes quoted whole arguments such as JSON.stringify("-modfile=/tmp/x").
  return /(?:^|[\s"'])-modfile(?:=|\s|$)/.test(flags);
}
