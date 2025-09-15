#!/usr/bin/env python3
"""
Setup script for nanostore Python bindings
"""

from setuptools import setup, find_packages
from setuptools.command.build_py import build_py
from setuptools.command.install import install
import os
import platform
import urllib.request
import tarfile
import zipfile
import json
from pathlib import Path


class DownloadLibraryCommand(build_py):
    """Custom command to download the shared library from GitHub releases"""
    
    def get_platform_info(self):
        """Get platform-specific information for downloading the right binary"""
        system = platform.system().lower()
        machine = platform.machine().lower()
        
        # Map platform names to goreleaser conventions
        os_map = {
            'linux': 'linux',
            'darwin': 'darwin',
            'windows': 'windows'
        }
        
        arch_map = {
            'x86_64': 'amd64',
            'amd64': 'amd64',
            'aarch64': 'arm64',
            'arm64': 'arm64'
        }
        
        if system not in os_map:
            raise RuntimeError(f"Unsupported operating system: {system}")
        
        if machine not in arch_map:
            raise RuntimeError(f"Unsupported architecture: {machine}")
        
        return os_map[system], arch_map[machine], system
    
    def get_latest_release_info(self):
        """Get the latest release information from GitHub API"""
        api_url = "https://api.github.com/repos/arthur-debert/nanostore/releases/latest"
        
        try:
            with urllib.request.urlopen(api_url) as response:
                return json.loads(response.read().decode())
        except Exception as e:
            # If we can't get the latest, try to read from a local VERSION file
            version_file = Path(__file__).parent.parent.parent.parent / "VERSION"
            if version_file.exists():
                version = version_file.read_text().strip()
                return {"tag_name": f"v{version}"}
            raise RuntimeError(f"Failed to get release info: {e}")
    
    def download_and_extract_library(self, download_url, target_dir):
        """Download and extract the shared library"""
        print(f"Downloading from {download_url}")
        
        # Create target directory
        os.makedirs(target_dir, exist_ok=True)
        
        # Download the archive
        archive_path = os.path.join(target_dir, "archive.tmp")
        urllib.request.urlretrieve(download_url, archive_path)
        
        # Extract based on file type
        if download_url.endswith('.tar.gz'):
            with tarfile.open(archive_path, 'r:gz') as tar:
                # Find the shared library file
                for member in tar.getmembers():
                    if member.name.endswith(('.so', '.dylib', '.dll')):
                        member.name = os.path.basename(member.name)
                        tar.extract(member, target_dir)
                        break
        elif download_url.endswith('.zip'):
            with zipfile.ZipFile(archive_path, 'r') as zip_file:
                # Find the shared library file
                for name in zip_file.namelist():
                    if name.endswith(('.so', '.dylib', '.dll')):
                        # Extract with just the filename
                        with zip_file.open(name) as source:
                            target_path = os.path.join(target_dir, os.path.basename(name))
                            with open(target_path, 'wb') as target:
                                target.write(source.read())
                        break
        
        # Clean up
        os.remove(archive_path)
    
    def run(self):
        """Download the appropriate shared library before building"""
        # Get platform info
        os_name, arch, system = self.get_platform_info()
        
        # Get release info
        release_info = self.get_latest_release_info()
        version = release_info['tag_name'].lstrip('v')
        
        # Construct download URL
        archive_name = f"nanostore-lib_{version}_{os_name}_{arch}"
        if system == 'windows':
            archive_name += ".zip"
        else:
            archive_name += ".tar.gz"
        
        download_url = f"https://github.com/arthur-debert/nanostore/releases/download/v{version}/{archive_name}"
        
        # Download to package directory
        target_dir = os.path.join(self.build_lib, 'nanostore')
        
        try:
            self.download_and_extract_library(download_url, target_dir)
            print(f"Successfully downloaded nanostore library v{version}")
        except Exception as e:
            print(f"Warning: Failed to download pre-built library: {e}")
            print("You may need to build the library manually")
        
        # Continue with normal build
        super().run()


# Read version from parent project
def get_version():
    """Get version from parent project VERSION file or default"""
    version_file = Path(__file__).parent.parent.parent.parent / "VERSION"
    if version_file.exists():
        return version_file.read_text().strip()
    return "0.1.0"


# Read README
def get_long_description():
    """Get long description from README"""
    readme_path = Path(__file__).parent / "README.md"
    if readme_path.exists():
        return readme_path.read_text()
    return "Python bindings for nanostore document store library"


setup(
    name="nanostore",
    version=get_version(),
    description="Python bindings for nanostore document store library",
    long_description=get_long_description(),
    long_description_content_type="text/markdown",
    author="Arthur Debert",
    author_email="arthur@debert.com",
    url="https://github.com/arthur-debert/nanostore",
    packages=find_packages(),
    package_data={
        'nanostore': ['*.so', '*.dylib', '*.dll'],
    },
    include_package_data=True,
    python_requires=">=3.7",
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Database",
        "Topic :: Software Development :: Libraries",
    ],
    keywords="document store, sqlite, id generation, todo, cli tools",
    cmdclass={
        'build_py': DownloadLibraryCommand,
    },
    extras_require={
        'dev': ['pytest', 'black', 'mypy', 'wheel', 'twine'],
        'test': ['pytest'],
    },
)