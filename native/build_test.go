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

package native_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot-native-image/native"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx   libcnb.BuildContext
		build native.Build
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "META-INF"), 0755)).To(Succeed())

		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "spring-graalvm-native",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
				},
			},
		}

		ctx.StackID = "test-stack-id"
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("contributes native image layer", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
Spring-Boot-Layers-Index: layers.idx
Start-Class: test-start-class
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(1))
		Expect(result.Layers[0].(native.NativeImage).Arguments).To(BeEmpty())
		Expect(result.Processes).To(ContainElements(
			libcnb.Process{Type: "native-image", Command: filepath.Join(ctx.Application.Path, "test-start-class"), Direct: true},
			libcnb.Process{Type: "task", Command: filepath.Join(ctx.Application.Path, "test-start-class"), Direct: true},
			libcnb.Process{Type: "web", Command: filepath.Join(ctx.Application.Path, "test-start-class"), Direct: true},
		))
	})

	context("BP_BOOT_NATIVE_IMAGE_BUILD_ARGUMENTS", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_BOOT_NATIVE_IMAGE_BUILD_ARGUMENTS", "test-native-image-argument")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_BOOT_NATIVE_IMAGE_BUILD_ARGUMENTS")).To(Succeed())
		})

		it("contributes native image build arguments", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
Spring-Boot-Layers-Index: layers.idx
Start-Class: test-start-class
`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers[0].(native.NativeImage).Arguments).To(Equal([]string{"test-native-image-argument"}))
		})

	})
}
