const { getDefaultConfig } = require("expo/metro-config");
const path = require("path");

const projectRoot = __dirname;
const workspaceRoot = path.resolve(projectRoot, "..");

const config = getDefaultConfig(projectRoot);

// mobile/ depends on shared/constraints via a `file:` dependency (symlinked
// into node_modules). Metro only watches/resolves within projectRoot by
// default, so it can't see files reached through that symlink unless the
// workspace root is added explicitly.
config.watchFolders = [workspaceRoot];
config.resolver.nodeModulesPaths = [
  path.resolve(projectRoot, "node_modules"),
  path.resolve(workspaceRoot, "node_modules"),
];
config.resolver.unstable_enableSymlinks = true;

module.exports = config;
