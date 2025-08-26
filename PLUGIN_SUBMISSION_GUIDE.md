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
- Open source technology (Yamcs)
- Non-commercial nature
- Apache-2.0 license
- Public GitHub repository
- Technology available for testing

### Technical Requirements
- Plugin metadata complete
- Comprehensive documentation
- Testing guide provided
- Build process working
- Screenshots included

### Known Issues

**Multi-Plugin Architecture Note:**
This plugin uses a multi-plugin architecture with nested components (datasource + panels). While the plugin validator flags this as an issue, this design provides a comprehensive Yamcs integration. We're submitting for review to get guidance from Grafana team on the best approach for this architecture.

- Nested plugin declarations (Architectural decision for Grafana review)
- GitHub URL rate limiting (temporary)

## Architecture Decision for Review

The plugin currently includes:
- **Main App Plugin**: Navigation and configuration pages
- **Datasource Plugin**: Yamcs server connectivity 
- **Panel Plugins**: Specialized visualization components

This architecture could be:
1. **Approved as-is** (multi-plugin bundle)
2. **Restructured** per Grafana guidance
3. **Split** into separate plugin submissions

We request Grafana team guidance on the preferred approach.

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
