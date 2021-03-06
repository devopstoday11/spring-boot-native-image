/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package native

import (
	"fmt"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

type Build struct {
	Logger bard.Logger
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	manifest, err := libjvm.NewManifest(context.Application.Path)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to read manifest in %s\n%w", context.Application.Path, err)
	}

	_, ok := manifest.Get("Spring-Boot-Version")
	if !ok {
		return libcnb.BuildResult{}, nil
	}

	b.Logger.Title(context.Buildpack)
	result := libcnb.NewBuildResult()

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	dc, err := libpak.NewDependencyCache(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
	}
	dc.Logger = b.Logger

	args, _ := cr.Resolve("BP_BOOT_NATIVE_IMAGE_BUILD_ARGUMENTS")

	dep, err := dr.Resolve("spring-graalvm-native", "")
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
	}

	n, err := NewNativeImage(context.Application.Path, args, dep, dc, manifest, context.StackID, result.Plan)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create native image layer\n%w", err)
	}
	n.Logger = b.Logger
	result.Layers = append(result.Layers, n)

	startClass, ok := manifest.Get("Start-Class")
	if !ok {
		return libcnb.BuildResult{}, fmt.Errorf("manifest does not contain Start-Class")
	}

	command := filepath.Join(context.Application.Path, startClass)
	result.Processes = append(result.Processes,
		libcnb.Process{Type: "native-image", Command: command, Direct: true},
		libcnb.Process{Type: "task", Command: command, Direct: true},
		libcnb.Process{Type: "web", Command: command, Direct: true},
	)

	return result, nil
}
