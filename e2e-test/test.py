#! /usr/bin/env python3

# Note: you'll need to have ucconfig in $PATH to run this script.

import filecmp
import os
import re
import subprocess
import sys
import tempfile

from contextlib import contextmanager
from pathlib import Path

TEST_DIR = os.path.dirname(__file__)

# https://stackoverflow.com/a/14693789
ANSI_ESCAPE_REGEXP = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
RESOURCE_DEFINITION_REGEXP = re.compile(r'TerraformTypeSuffix:\s+"([a-z0-9_]+)"')

@contextmanager
def log_step(message):
    if os.getenv("GITHUB_ACTIONS"):
        print(f'::group::{message}')
    else:
        print(f'\033[93m{message}\033[00m')
    try:
        yield
    finally:
        if os.getenv("GITHUB_ACTIONS"):
            print('::endgroup::')

class ChildProcessError(Exception):
    def __init__(self, output):
        self.output = output

def run_capturing_output(cmd):
    output = ''
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    for line in process.stdout:
        decoded = line.decode('utf-8')
        # Keep color codes in the output to the terminal, but strip them for the
        # captured output string so that we can search that string without color
        # codes messing up the match
        print(decoded.rstrip('\n'))
        output += ANSI_ESCAPE_REGEXP.sub('', decoded)
    if process.wait() != 0:
        raise ChildProcessError(output)
    return output

def apply_manifest(manifest_name):
    return run_capturing_output(['ucconfig', 'apply', os.path.join(TEST_DIR, manifest_name), '--auto-approve'])

def assert_no_changes(manifest_name):
    out = run_capturing_output(['ucconfig', 'apply', os.path.join(TEST_DIR, manifest_name), '--dry-run'])
    if 'No changes. Your infrastructure matches the configuration.' not in out:
        raise Exception('Got an unexpected diff!')
    return out

def assert_genmanifest_matches(manifest_name):
    with tempfile.TemporaryDirectory() as tmpdirname:
        run_capturing_output(['ucconfig', 'gen-manifest', os.path.join(tmpdirname, manifest_name)])
        diff_found = False

        # Compare manifest file
        if not filecmp.cmp(os.path.join(TEST_DIR, manifest_name), os.path.join(tmpdirname, manifest_name)):
            print('Running gen-manifest modified the manifest:')
            subprocess.run(['diff', '-u', os.path.join(TEST_DIR, manifest_name), os.path.join(tmpdirname, manifest_name)])
            diff_found = True

        # Compare values files
        values_dirname = Path(manifest_name).stem + '_values'
        existing_values_dir = os.path.join(TEST_DIR, values_dirname)
        generated_values_dir = os.path.join(tmpdirname, values_dirname)
        values_cmp = filecmp.dircmp(existing_values_dir, generated_values_dir)
        for f in values_cmp.left_only:
            print(f'Running genmanifest deleted {values_dirname}/{f}')
            diff_found = True
        for f in values_cmp.right_only:
            print(f'Running genmanifest created {values_dirname}/{f}')
            diff_found = True
        for f in values_cmp.diff_files:
            print(f'Running genmanifest modified {values_dirname}/{f}:')
            subprocess.run(['diff', '-u', os.path.join(existing_values_dir, f), os.path.join(generated_values_dir, f)])
            print('') # add blank line after diff output as separation
            diff_found = True

        if diff_found:
            raise Exception('genmanifest produced a differing manifest and/or external value files')

def list_resource_types():
    with open(os.path.join(TEST_DIR, '../internal/resourcetypes/types.go')) as f:
        return RESOURCE_DEFINITION_REGEXP.findall(f.read())

def main():
    # A previous failed run could have left unwanted resources. Start by
    # deleting all non-system resources.
    with log_step("Applying empty.yaml to get to baseline state..."):
        apply_manifest('empty.yaml')
    with log_step("Applying empty.yaml again should not change anything..."):
        assert_no_changes('empty.yaml')

    # Apply config with a bunch of resources
    with log_step("Applying lots-of-resources.yaml to test resource creation..."):
        output = apply_manifest('lots-of-resources.yaml')
        for tf_type_suffix in list_resource_types():
            if not re.search(f'userclouds_{tf_type_suffix}' + r'\.[a-zA-Z0-9_-]+ will be created', output):
                raise Exception(f'Did not see resource type {tf_type_suffix} created in the output. Please add test coverage')
    with log_step("Applying lots-of-resources.yaml again should not change anything..."):
        assert_no_changes('lots-of-resources.yaml')
    with log_step("Regenerating lots-of-resources.yaml should generate the same manifest..."):
        assert_genmanifest_matches('lots-of-resources.yaml')

    with log_step("Applying lots-of-resources-modified.yaml to test resource modification..."):
        output = apply_manifest('lots-of-resources-modified.yaml')
        for tf_type_suffix in list_resource_types():
            if not re.search(f'userclouds_{tf_type_suffix}' + r'\.[a-zA-Z0-9_-]+ (?:will be updated|must be replaced)', output):
                raise Exception(f'Did not see resource type {tf_type_suffix} updated in the output. Please add test coverage')
    with log_step("Applying lots-of-resources-modified.yaml again should not change anything..."):
        assert_no_changes('lots-of-resources-modified.yaml')
    with log_step("Regenerating lots-of-resources-modified.yaml should generate the same manifest..."):
        assert_genmanifest_matches('lots-of-resources-modified.yaml')

    # Apply empty config
    with log_step("Applying empty.yaml to test resource deletion..."):
        apply_manifest('empty.yaml')
    with log_step("Applying empty.yaml again should not change anything..."):
        assert_no_changes('empty.yaml')

if __name__ == '__main__':
    main()
