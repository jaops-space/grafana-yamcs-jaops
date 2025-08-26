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

### ✅ Community Plugin Requirements
- [x] Open source technology (Yamcs)
- [x] Non-commercial nature
- [x] Apache-2.0 license
- [x] Public GitHub repository
- [x] Technology available for testing

### ✅ Technical Requirements
- [x] Plugin metadata complete
- [x] Comprehensive documentation
- [x] Testing guide provided
- [x] Build process working
- [x] Screenshots included

### ⚠️ Known Issues
- [ ] Nested plugin declarations need to be added to main plugin.json
- [ ] GitHub URL rate limiting (temporary)

## Submission Package Contents

1. **Plugin ZIP**: Signed plugin archive
2. **Source Code**: GitHub repository
3. **Documentation**: 
   - README.md
   - TESTING_GUIDE.md
   - setup_instructions.md
4. **Screenshots**: High-quality interface screenshots
5. **Test Environment**: Provisioning configuration

## Next Steps

1. Fix nested plugin declarations
2. Test signing workflow
3. Create GitHub release
4. Submit to Grafana for review

## Support Information

- **Author**: JAOPS
- **Email**: info@jaops.org
- **Website**: https://www.jaops.com
- **Repository**: https://github.com/jaops-space/grafana-yamcs-jaops
