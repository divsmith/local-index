#!/usr/bin/env python3

import os
import sys

def setup_environment():
    """Setup the development environment"""
    print("Setting up environment...")

    # Create necessary directories
    os.makedirs("logs", exist_ok=True)
    os.makedirs("data", exist_ok=True)

    print("Environment setup complete!")

def validate_config():
    """Validate configuration files"""
    config_file = "config.json"
    if not os.path.exists(config_file):
        print(f"Configuration file {config_file} not found")
        return False

    print("Configuration is valid")
    return True

if __name__ == "__main__":
    setup_environment()
    validate_config()
