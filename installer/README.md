# Delta CLI Windows Installer

This directory contains the Windows installer configuration for Delta CLI.

## Requirements

To build the Windows installer, you need one of the following:

1. **Windows**: [Inno Setup 6](https://jrsoftware.org/isinfo.php)
2. **Linux/macOS**: [Wine](https://www.winehq.org/) + Inno Setup

## Building the Installer

1. First, build the Windows executable:
   ```bash
   make build TARGET_OS=windows TARGET_ARCH=amd64
   ```

2. Build the installer:
   ```bash
   make installer
   ```

The installer will be created at: `build/installer/delta-setup-{version}.exe`

## Installer Features

- Installs Delta CLI to Program Files
- Optional: Adds Delta to system PATH
- Optional: Creates desktop shortcut
- Includes all necessary files (i18n, templates, patterns)
- Uninstaller included

## Icon Generation

The `create_icon.py` script generates the Delta icon. Requirements:
```bash
pip install Pillow
```

To regenerate the icon:
```bash
python3 create_icon.py
```

## Customization

Edit `delta-installer.iss` to customize:
- Installation paths
- Included files
- Setup behavior
- UI strings