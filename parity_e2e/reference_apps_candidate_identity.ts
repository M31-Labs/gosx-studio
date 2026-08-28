import { execFileSync } from "node:child_process";
import {
  chmodSync,
  copyFileSync,
  existsSync,
  lstatSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  realpathSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";
import { randomUUID } from "node:crypto";

export const STUDIO_MODULE_PATH = "m31labs.dev/gosx-studio";
export const CANDIDATE_REPO_ENV = "GOSX_STUDIO_CANDIDATE_REPO";
export const CANDIDATE_SHA_ENV = "GOSX_STUDIO_CANDIDATE_SHA";

export type CandidateIdentity = {
  candidateRepo: string;
  candidateSHA: string;
};

export type GoModuleGraph = {
  Path?: string;
  Dir?: string;
  Replace?: {
    Path?: string;
    Dir?: string;
  };
};

export type CandidateSourceCopy = {
  /** The untouched reference-app checkout used as the copy source. */
  hostRepo: string;
  /** A unique sibling copy whose go.mod contains the candidate replacement. */
  sourceRepo: string;
  /** Remove only this helper-owned copy; safe to call more than once. */
  dispose: () => void;
};

const excludedDirectoryNames = new Set([
  ".ferrous-wheel-build",
  ".git",
  ".graft",
  ".gosx",
  ".tiller",
  ".tmp",
  "build",
  "cache",
  "data",
  "dist",
  "node_modules",
  "tmp",
]);

const safeEnvironmentExamples = new Set([".env.example", ".env.sample", ".env.template"]);

/**
 * Read and validate the explicit candidate opt-in used by reference-app CI.
 *
 * Local runs may set only GOSX_STUDIO_CANDIDATE_REPO, in which case the SHA is
 * read from the candidate checkout. CI must also provide the expected SHA so
 * a green run cannot silently move to another checkout between workflow steps.
 */
export function loadCandidateIdentity(env: NodeJS.ProcessEnv = process.env): CandidateIdentity | null {
  const configuredRepo = env[CANDIDATE_REPO_ENV]?.trim();
  if (!configuredRepo) return null;

  const candidateRepo = resolveDirectory(configuredRepo, `${CANDIDATE_REPO_ENV} candidate repository`);
  const modulePath = readModulePath(path.join(candidateRepo, "go.mod"));
  if (modulePath !== STUDIO_MODULE_PATH) {
    throw new Error(
      `${CANDIDATE_REPO_ENV} must contain module ${STUDIO_MODULE_PATH}; found ${modulePath} in ${candidateRepo}`,
    );
  }

  const expectedSHA = env[CANDIDATE_SHA_ENV]?.trim();
  if (isCI(env) && !expectedSHA) {
    throw new Error(`${CANDIDATE_SHA_ENV} is required when ${CANDIDATE_REPO_ENV} is set in CI`);
  }
  if (expectedSHA) validateSHA(expectedSHA, CANDIDATE_SHA_ENV);

  const candidateSHA = readGitHead(candidateRepo);
  if (expectedSHA) assertCandidateSHA(expectedSHA, candidateSHA);

  return { candidateRepo, candidateSHA };
}

/**
 * Assert the exact module graph shape required for a candidate run.
 *
 * Go's JSON field names are represented directly here so the failure names
 * the same Module.Replace.Dir value that CI needs to prove. A local replace's
 * Replace.Path is a filesystem path in Go's output, so identity is checked by
 * the canonical replacement directory rather than by that field.
 */
export function assertResolvedCandidateModule(
  module: GoModuleGraph,
  identity: CandidateIdentity,
): void {
  if (module.Path !== STUDIO_MODULE_PATH) {
    throw new Error(
      `go module graph selected ${module.Path || "<missing module path>"}; expected ${STUDIO_MODULE_PATH}`,
    );
  }

  const replacementDir = module.Replace?.Dir;
  if (!replacementDir) {
    throw new Error(
      `go module graph did not expose Module.Replace.Dir for ${STUDIO_MODULE_PATH}; refusing to test a published module`,
    );
  }

  const resolvedReplacementDir = resolveDirectory(replacementDir, "Module.Replace.Dir");
  if (resolvedReplacementDir !== identity.candidateRepo) {
    throw new Error(
      `Module.Replace.Dir resolved to ${resolvedReplacementDir}; expected checked-out candidate ${identity.candidateRepo}`,
    );
  }
}

export function assertCandidateSHA(expectedSHA: string, actualSHA: string): void {
  validateSHA(expectedSHA, "expected candidate SHA");
  validateSHA(actualSHA, "resolved candidate SHA");
  if (expectedSHA.toLowerCase() !== actualSHA.toLowerCase()) {
    throw new Error(
      `candidate SHA mismatch: workflow expected ${expectedSHA}, checkout resolved ${actualSHA}`,
    );
  }
}

/**
 * Prepare a candidate run in a helper-owned sibling copy of a reference app.
 *
 * The GoSX CLI deliberately replaces inherited GOFLAGS with its own module
 * flags, so an alternate -modfile cannot reliably reach the CLI's internal
 * go list/build/WASM subprocesses. A source copy gives both `gosx build` and
 * `go run` the same candidate-bearing canonical go.mod while leaving the real
 * checkout byte-for-byte untouched. The copy is a sibling of the host (rather
 * than a child of a random temp directory), which preserves relative replace
 * paths such as ../shared-module; source-local relative replacements continue
 * to resolve inside the copied tree.
 *
 * Only tracked and non-ignored working-tree files are copied when the host is
 * a Git checkout. The fallback recursive copier is for synthetic test hosts.
 * Runtime state, caches, secrets, and generated output are excluded in both
 * modes. The caller owns the returned copy and must dispose it after use.
 */
export function createCandidateSourceCopy(
  hostRepo: string,
  identity: CandidateIdentity,
): CandidateSourceCopy {
  const sourceRoot = resolveDirectory(hostRepo, "reference app repository");
  const sourceModfile = path.join(sourceRoot, "go.mod");
  if (!existsSync(sourceModfile)) {
    throw new Error(`reference app repository has no go.mod: ${sourceRoot}`);
  }

  const source = readFileSync(sourceModfile, "utf8");
  if (hasStudioReplacement(source)) {
    throw new Error(
      `reference app go.mod already declares a ${STUDIO_MODULE_PATH} replacement; refusing ambiguous candidate resolution`,
    );
  }

  const sourceRepo = path.join(
    path.dirname(sourceRoot),
    `.${path.basename(sourceRoot)}.gosx-studio-candidate-${process.pid}-${randomUUID()}`,
  );

  mkdirSync(sourceRepo);
  try {
    copyWorkingTree(sourceRoot, sourceRepo);
    const copiedModfile = path.join(sourceRepo, "go.mod");
    if (!existsSync(copiedModfile)) {
      throw new Error(`candidate source copy did not contain go.mod: ${sourceRepo}`);
    }
    const copiedSource = readFileSync(copiedModfile, "utf8");
    const body = `${copiedSource.trimEnd()}

// CI-only candidate source copy; removed by the harness after the run.
replace ${STUDIO_MODULE_PATH} => ${quoteGoString(identity.candidateRepo)}
`;
    writeFileSync(copiedModfile, body, "utf8");
  } catch (error) {
    rmSync(sourceRepo, { force: true, recursive: true });
    throw error;
  }

  let disposed = false;
  return {
    hostRepo: sourceRoot,
    sourceRepo,
    dispose: () => {
      if (disposed) return;
      disposed = true;
      rmSync(sourceRepo, { force: true, recursive: true });
    },
  };
}

/**
 * Build a fresh command environment for the actual source copy or checkout.
 * Caller-provided PORT/data values are preserved; no environment from another
 * server lease is cached here. An existing -modfile is rejected because it
 * could silently bypass the candidate replacement in the copied go.mod.
 */
export function withCandidateModuleEnvironment(baseEnv: NodeJS.ProcessEnv): NodeJS.ProcessEnv {
  const existingGoFlags = baseEnv.GOFLAGS?.trim() ?? "";
  if (hasModfileFlag(existingGoFlags)) {
    throw new Error("GOFLAGS already contains -modfile; refusing to bypass the candidate source copy");
  }
  return {
    ...baseEnv,
    GOWORK: "off",
  };
}

/** Resolve the module graph in the exact source directory passed to Go. */
export function resolveCandidateModuleGraph(
  sourceRepo: string,
  baseEnv: NodeJS.ProcessEnv = process.env,
): GoModuleGraph {
  const sourceRoot = resolveDirectory(sourceRepo, "candidate source repository");
  const result = execFileSync(
    "go",
    ["list", "-m", "-json", STUDIO_MODULE_PATH],
    {
      cwd: sourceRoot,
      env: withCandidateModuleEnvironment(baseEnv),
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    },
  );
  try {
    return JSON.parse(result) as GoModuleGraph;
  } catch (error) {
    throw new Error(`go list returned invalid module graph JSON: ${String(error)}`);
  }
}

function readModulePath(modfile: string): string {
  let source: string;
  try {
    source = readFileSync(modfile, "utf8");
  } catch (error) {
    throw new Error(`cannot read candidate go.mod ${modfile}: ${String(error)}`);
  }
  const match = /^\s*module\s+(\S+)\s*$/m.exec(source);
  if (!match) throw new Error(`candidate go.mod has no module declaration: ${modfile}`);
  return match[1];
}

function readGitHead(repo: string): string {
  try {
    const sha = execFileSync("git", ["rev-parse", "--verify", "HEAD"], {
      cwd: repo,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    }).trim();
    validateSHA(sha, "resolved candidate SHA");
    return sha;
  } catch (error) {
    throw new Error(`cannot resolve candidate checkout HEAD in ${repo}: ${String(error)}`);
  }
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

function validateSHA(value: string, label: string): void {
  if (!/^[0-9a-f]{40}$/i.test(value)) {
    throw new Error(`${label} must be a full 40-character git SHA; found ${value || "<empty>"}`);
  }
}

function isCI(env: NodeJS.ProcessEnv): boolean {
  return /^(?:1|true)$/i.test(env.CI?.trim() ?? "");
}

function hasStudioReplacement(source: string): boolean {
  const escapedModulePath = STUDIO_MODULE_PATH.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(
    `(?:^|\\n)\\s*(?:${escapedModulePath}(?:\\s+v[^\\s]+)?\\s*=>|replace\\s+${escapedModulePath}(?:\\s+v[^\\s]+)?\\s*=>)`,
    "m",
  ).test(source);
}

function hasModfileFlag(flags: string): boolean {
  return /(?:^|[\s"'])-modfile(?:=|\s|$)/.test(flags);
}

function quoteGoString(value: string): string {
  return JSON.stringify(value);
}

function copyWorkingTree(sourceRoot: string, destinationRoot: string): void {
  const gitFiles = listGitWorkingTreeFiles(sourceRoot);
  if (gitFiles) {
    for (const relativeFile of gitFiles) copyWorkingTreeFile(sourceRoot, destinationRoot, relativeFile);
    return;
  }
  copyDirectory(sourceRoot, destinationRoot, "");
}

function listGitWorkingTreeFiles(sourceRoot: string): string[] | null {
  try {
    const output = execFileSync(
      "git",
      ["ls-files", "--cached", "--others", "--exclude-standard", "-z"],
      {
        cwd: sourceRoot,
        encoding: "utf8",
        stdio: ["ignore", "pipe", "ignore"],
      },
    );
    return output.split("\0").filter(Boolean);
  } catch {
    return null;
  }
}

function copyDirectory(sourceDir: string, destinationDir: string, relativeDir: string): void {
  for (const entry of readdirSync(sourceDir, { withFileTypes: true })) {
    const relativePath = relativeDir ? path.join(relativeDir, entry.name) : entry.name;
    if (shouldSkipCandidatePath(relativePath, entry.isDirectory())) continue;
    const sourcePath = path.join(sourceDir, entry.name);
    const destinationPath = path.join(destinationDir, entry.name);
    if (entry.isDirectory()) {
      mkdirSync(destinationPath, { recursive: true });
      copyDirectory(sourcePath, destinationPath, relativePath);
    } else {
      copyWorkingTreeEntry(sourcePath, destinationPath);
    }
  }
}

function copyWorkingTreeFile(sourceRoot: string, destinationRoot: string, relativeFile: string): void {
  const normalized = relativeFile.replaceAll("/", path.sep);
  if (path.isAbsolute(normalized) || normalized === ".." || normalized.startsWith(`..${path.sep}`)) {
    throw new Error(`git returned an unsafe source path: ${relativeFile}`);
  }
  if (shouldSkipCandidatePath(normalized, false)) return;
  const sourcePath = path.join(sourceRoot, normalized);
  const destinationPath = path.join(destinationRoot, normalized);
  const parent = path.dirname(destinationPath);
  mkdirSync(parent, { recursive: true });
  copyWorkingTreeEntry(sourcePath, destinationPath);
}

function copyWorkingTreeEntry(sourcePath: string, destinationPath: string): void {
  const info = lstatSync(sourcePath);
  if (info.isSymbolicLink()) {
    // A symlink can escape the copied tree even when its target is relative;
    // rejecting all links is safer for CI and the reference apps have no
    // required source links once node_modules and caches are excluded.
    throw new Error(`refusing symlink in candidate source copy: ${sourcePath}`);
  }
  if (!info.isFile()) return;
  copyFileSync(sourcePath, destinationPath);
  chmodSync(destinationPath, info.mode & 0o7777);
}

function shouldSkipCandidatePath(relativePath: string, directory: boolean): boolean {
  const components = relativePath.split(/[\\/]+/).filter(Boolean);
  if (components.some((component) => excludedDirectoryNames.has(component))) return true;
  const basename = components.at(-1) ?? "";
  if (directory && excludedDirectoryNames.has(basename)) return true;
  if (basename === ".env" || (basename.startsWith(".env.") && !safeEnvironmentExamples.has(basename))) return true;
  if (/\.(?:db|sqlite|sqlite3|sqlite-wal|sqlite-shm)$/i.test(basename)) return true;
  return false;
}
