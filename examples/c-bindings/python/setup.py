#!/usr/bin/env python3
"""
Setup script for nanostore Python bindings
"""

from setuptools import setup, find_packages
import os
import subprocess
import shutil
from distutils.command.build import build as _build
from setuptools.command.install import install as _install


class BuildCommand(_build):
    """Custom build command that compiles the Go library"""
    
    def run(self):
        # Build the Go library
        print("Building Go library...")
        result = subprocess.run([
            'go', 'build', '-buildmode=c-shared', 
            '-o', 'libnanostore.so', 'main.go'
        ], cwd='..', capture_output=True, text=True)
        
        if result.returncode != 0:
            print("Go build failed:")
            print(result.stderr)
            raise RuntimeError("Failed to build Go library")
        
        # Copy library to package directory
        src_lib = '../libnanostore.so'
        dst_lib = 'nanostore/libnanostore.so'
        os.makedirs('nanostore', exist_ok=True)
        shutil.copy2(src_lib, dst_lib)
        print(f"Copied {src_lib} to {dst_lib}")
        
        # Continue with normal build
        _build.run(self)


class InstallCommand(_install):
    """Custom install command"""
    
    def run(self):
        self.run_command('build')
        _install.run(self)


setup(
    name="nanostore",
    version="0.3.0",
    description="Python bindings for nanostore document store library",
    long_description=open("../../../README.md").read(),
    long_description_content_type="text/markdown",
    author="Arthur Debert",
    author_email="arthur@debert.com",
    url="https://github.com/arthur-debert/nanostore",
    packages=find_packages(),
    package_data={
        'nanostore': ['libnanostore.so'],
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
        'build': BuildCommand,
        'install': InstallCommand,
    },
    extras_require={
        'dev': ['pytest', 'black', 'mypy'],
        'test': ['pytest'],
    },
)