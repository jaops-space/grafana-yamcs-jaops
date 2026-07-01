# Plugin Submission Guide for Grafana Marketplace

This document outlines the steps and requirements for submitting the JAOPS Yamcs plugin to the Grafana marketplace.

## Plugin Overview

The JAOPS Yamcs plugin is a comprehensive integration that provides:
- **App Plugin**: Main application with navigation pages
- **Datasource Plugin**: Connects to Yamcs servers for data retrieval
- **Panel Plugins**: Specialized visualization and commanding panels
  - Commanding Panel
  - Command History Panel
  - Image Panel
  - Static Image Panel
  - Variable Commanding Panel

## Plugin Structure

This is a multi-plugin package containing:
```
jaops-yamcs-app/                    # Main app plugin
├── datasource/                     # Yamcs datasource plugin
├── commanding-panel/               # Commanding panel plugin
├── command-history-panel/          # Command history panel plugin
├── image-panel/                    # Image panel plugin
├── static-image-panel/             # Static image panel plugin
├── variable-commanding-panel/      # Variable commanding panel plugin
└── screenshots/                    # Plugin screenshots
```

## Compliance Checklist

### Community Plugin Requirements
- Interfacing with open source technology (Yamcs: www.yamcs.org, AGPL-3.0 license)
- Plugin is MIT license
- Public GitHub repository
- Technology is available for testing (with examples)
- Non-commercial nature

### Technical Requirements
- Plugin metadata complete
- Keywords for discoverability
- Author information and contact details
- Comprehensive documentation
- Testing guide provided
- Build process working
- Screenshots included
- Plugin validator passing

## Submission Package Contents

1. **Plugin ZIP**: Signed plugin archive
2. **Source Code**: GitHub repository
3. **Documentation**: 
   - README.md
   - TESTING_GUIDE.md
   - setup_instructions.md
4. **Test Environment**: Provisioning configuration

5. **Security Audit and submission steps**
```bash
# security audit
# critical and high vulnerabilities must be fixed before submission (CVSS >=7), low and medium can be ignored.
# gotcha: upgrading the SDK can raise the required Go version in go.mod. if so, bump the pinned
#         go-version in .github/workflows/ci.yml and release.yml to match (CI uses GOTOOLCHAIN=local, so it won't auto-download).
pnpm audit
osv-scanner --recursive .

# version bump: set the new version in package.json (single source of truth,
# injected into each plugin.json as %VERSION% at build time). the git tag must match.
# e.g. bump "version" in package.json to 1.0.5

# functional test
mage build:backend
pnpm run build
# manual step: test demo dashboard works well
# pnpm run server

# package, validate and submit
mkdir -p dist/screenshots && cp screenshots/*.png dist/screenshots/
rm -f jaops-yamcs-app.zip && rm -rf /tmp/jaops-yamcs-app && cp -r dist /tmp/jaops-yamcs-app && (cd /tmp && zip -r jaops-yamcs-app.zip jaops-yamcs-app) && mv /tmp/jaops-yamcs-app.zip .

npx @grafana/plugin-validator@latest jaops-yamcs-app.zip
# ignore sponsorship related warnings 
# ignore MANIFEST.md warning (the signing is done in the CI)

md5sum jaops-yamcs-app.zip

# commit and tag: tag must match package.json version. pushing the tag triggers the CI release (signing + artifact).
git commit -am "chore: release v1.0.5"
git tag v1.0.5
git push origin <branch> v1.0.5
```


