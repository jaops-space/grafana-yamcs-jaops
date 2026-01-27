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

5. **Security Audit and sumbission steps**
```bash
# security audit
pnpm audit
osv-scanner --recursive .

# functional test
mage build:backend
pnpm run build
# manual step: test demo dashboard works well

# package, validate and submit
mkdir -p dist/screenshots && cp screenshots/*.png dist/screenshots/
rm -f jaops-yamcs-app.zip && rm -rf /tmp/jaops-yamcs-app && cp -r dist /tmp/jaops-yamcs-app && (cd /tmp && zip -r jaops-yamcs-app.zip jaops-yamcs-app) && mv /tmp/jaops-yamcs-app.zip .

npx @grafana/plugin-validator@latest jaops-yamcs-app.zip

md5sum jaops-yamcs-app.zip
```


