// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package commonsteps

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

const TestFixtures = "test-fixtures"

func TestStepCreateFloppy_Impl(t *testing.T) {
	var raw interface{}
	raw = new(StepCreateFloppy)
	if _, ok := raw.(multistep.Step); !ok {
		t.Fatalf("StepCreateFloppy should be a step")
	}
}

func testStepCreateFloppyState(t *testing.T) multistep.StateBag {
	state := new(multistep.BasicStateBag)
	state.Put("ui", &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	})
	return state
}

func TestStepCreateFloppy(t *testing.T) {
	state := testStepCreateFloppyState(t)
	step := new(StepCreateFloppy)

	dir, err := ioutil.TempDir("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)

	count := 10
	expected := count
	files := make([]string, count)

	prefix := "exists"
	ext := ".tmp"

	for i := 0; i < expected; i++ {
		files[i] = path.Join(dir, prefix+strconv.Itoa(i)+ext)

		_, err := os.Create(files[i])
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	lists := [][]string{
		files,
		{dir + string(os.PathSeparator) + prefix + "*" + ext},
		{dir + string(os.PathSeparator) + prefix + "?" + ext},
		{dir + string(os.PathSeparator) + prefix + "[0123456789]" + ext},
		{dir + string(os.PathSeparator) + prefix + "[0-9]" + ext},
		{dir + string(os.PathSeparator)},
		{dir},
	}

	for _, step.Files = range lists {
		if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
			t.Fatalf("bad action: %#v for %v", action, step.Files)
		}

		if _, ok := state.GetOk("error"); ok {
			t.Fatalf("state should be ok for %v", step.Files)
		}

		floppy_path := state.Get("floppy_path").(string)

		if _, err := os.Stat(floppy_path); err != nil {
			t.Fatalf("file not found: %s for %v", floppy_path, step.Files)
		}

		if len(step.FilesAdded) != expected {
			t.Fatalf("expected %d, found %d for %v", expected, len(step.FilesAdded), step.Files)
		}

		step.Cleanup(state)

		if _, err := os.Stat(floppy_path); err == nil {
			t.Fatalf("file found: %s for %v", floppy_path, step.Files)
		}
	}
}

func TestStepCreateFloppy_missing(t *testing.T) {
	state := testStepCreateFloppyState(t)
	step := new(StepCreateFloppy)

	dir, err := ioutil.TempDir("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)

	count := 2
	expected := 0
	files := make([]string, count)

	prefix := "missing"

	for i := 0; i < count; i++ {
		files[i] = path.Join(dir, prefix+strconv.Itoa(i))
	}

	lists := [][]string{
		files,
	}

	for _, step.Files = range lists {
		if action := step.Run(context.Background(), state); action != multistep.ActionHalt {
			t.Fatalf("bad action: %#v for %v", action, step.Files)
		}

		if _, ok := state.GetOk("error"); !ok {
			t.Fatalf("state should not be ok for %v", step.Files)
		}

		floppy_path := state.Get("floppy_path")

		if floppy_path != nil {
			t.Fatalf("floppy_path is not nil for %v", step.Files)
		}

		if len(step.FilesAdded) != expected {
			t.Fatalf("expected %d, found %d for %v", expected, len(step.FilesAdded), step.Files)
		}
	}
}

func TestStepCreateFloppy_notfound(t *testing.T) {
	state := testStepCreateFloppyState(t)
	step := new(StepCreateFloppy)

	dir, err := ioutil.TempDir("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)

	count := 2
	expected := 0
	files := make([]string, count)

	prefix := "notfound"

	for i := 0; i < count; i++ {
		files[i] = path.Join(dir, prefix+strconv.Itoa(i))
	}

	lists := [][]string{
		{dir + string(os.PathSeparator) + prefix + "*"},
		{dir + string(os.PathSeparator) + prefix + "?"},
		{dir + string(os.PathSeparator) + prefix + "[0123456789]"},
		{dir + string(os.PathSeparator) + prefix + "[0-9]"},
		{dir + string(os.PathSeparator)},
		{dir},
	}

	for _, step.Files = range lists {
		if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
			t.Fatalf("bad action: %#v for %v", action, step.Files)
		}

		if _, ok := state.GetOk("error"); ok {
			t.Fatalf("state should be ok for %v", step.Files)
		}

		floppy_path := state.Get("floppy_path").(string)

		if _, err := os.Stat(floppy_path); err != nil {
			t.Fatalf("file not found: %s for %v", floppy_path, step.Files)
		}

		if len(step.FilesAdded) != expected {
			t.Fatalf("expected %d, found %d for %v", expected, len(step.FilesAdded), step.Files)
		}

		step.Cleanup(state)

		if _, err := os.Stat(floppy_path); err == nil {
			t.Fatalf("file found: %s for %v", floppy_path, step.Files)
		}
	}
}

func TestStepCreateFloppyDirectories(t *testing.T) {
	const TestName = "floppy-hier"

	// file-system hierarchies
	var basePath = filepath.Join(".", TestFixtures, TestName)

	type contentsTest struct {
		dirs   []string
		result []string
	}

	// keep in mind that .FilesAdded doesn't keep track of the target filename or directories, but rather the source filename.
	directories := [][]contentsTest{
		{
			{dirs: []string{"file1", "file2", "file3"}, result: []string{"file1", "file2", "file3"}},
			{dirs: []string{"file?"}, result: []string{"file1", "file2", "file3"}},
			{dirs: []string{"*"}, result: []string{"file1", "file2", "file3"}},
		},
		{
			{dirs: []string{"dir1"}, result: []string{"dir1/file1", "dir1/file2", "dir1/file3"}},
			{dirs: []string{"dir1/file1", "dir1/file2", "dir1/file3"}, result: []string{"dir1/file1", "dir1/file2", "dir1/file3"}},
			{dirs: []string{"*"}, result: []string{"dir1/file1", "dir1/file2", "dir1/file3"}},
			{dirs: []string{"*/*"}, result: []string{"dir1/file1", "dir1/file2", "dir1/file3"}},
		},
		{
			{dirs: []string{"dir1"}, result: []string{"dir1/file1", "dir1/subdir1/file1", "dir1/subdir1/file2"}},
			{dirs: []string{"dir2/*"}, result: []string{"dir2/subdir1/file1", "dir2/subdir1/file2"}},
			{dirs: []string{"dir2/subdir1"}, result: []string{"dir2/subdir1/file1", "dir2/subdir1/file2"}},
			{dirs: []string{"dir?"}, result: []string{"dir1/file1", "dir1/subdir1/file1", "dir1/subdir1/file2", "dir2/subdir1/file1", "dir2/subdir1/file2"}},
		},
	}

	// create the hierarchy for each file
	for i := 0; i < 2; i++ {
		dir := filepath.Join(basePath, fmt.Sprintf("test-%d", i))

		for _, test := range directories[i] {
			// create a new state and step
			state := testStepCreateFloppyState(t)
			step := new(StepCreateFloppy)

			// modify step.Directories with ones from testcase
			step.Directories = []string{}
			for _, c := range test.dirs {
				step.Directories = append(step.Directories, filepath.Join(dir, filepath.FromSlash(c)))
			}
			log.Println(fmt.Sprintf("Trying against floppy_dirs : %v", step.Directories))

			// run the step
			if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
				t.Fatalf("bad action: %#v for %v : %v", action, step.Directories, state.Get("error"))
			}

			if _, ok := state.GetOk("error"); ok {
				t.Fatalf("state should be ok for %v : %v", step.Directories, state.Get("error"))
			}

			floppy_path := state.Get("floppy_path").(string)
			if _, err := os.Stat(floppy_path); err != nil {
				t.Fatalf("file not found: %s for %v : %v", floppy_path, step.Directories, err)
			}

			// check the FilesAdded array to see if it matches
			for _, rpath := range test.result {
				fpath := filepath.Join(dir, filepath.FromSlash(rpath))
				if !step.FilesAdded[fpath] {
					t.Fatalf("unable to find file: %s for %v", fpath, step.Directories)
				}
			}

			// cleanup the step
			step.Cleanup(state)

			if _, err := os.Stat(floppy_path); err == nil {
				t.Fatalf("file found: %s for %v", floppy_path, step.Directories)
			}
		}
	}
}

func TestStepCreateFloppyContent(t *testing.T) {
	// create a new state and step
	state := testStepCreateFloppyState(t)
	step := new(StepCreateFloppy)

	step.Content = map[string]string{
		"subfolder/meta-data": "subfolder/meta-data from content",
		"user-data":           "user-data from content",
	}

	// run the step
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v for %v : %v", action, step.Content, state.Get("error"))
	}

	if _, ok := state.GetOk("error"); ok {
		t.Fatalf("state should be ok for %v : %v", step.Content, state.Get("error"))
	}

	floppy_path := state.Get("floppy_path").(string)
	if _, err := os.Stat(floppy_path); err != nil {
		t.Fatalf("file not found: %s for %v : %v", floppy_path, step.Content, err)
	}

	// check the FilesAdded array to see if it matches
	for path := range step.Content {
		if !step.FilesAdded[path] {
			t.Fatalf("unable to find file: %s for %v", path, step.Content)
		}
	}

	// cleanup the step
	step.Cleanup(state)

	if _, err := os.Stat(floppy_path); err == nil {
		t.Fatalf("file found: %s for %v", floppy_path, step.Content)
	}
}
