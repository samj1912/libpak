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

package libpak_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

func testLayer(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layer libcnb.Layer
	)

	it.Before(func() {
		var err error

		layer.Path, err = ioutil.TempDir("", "layer")
		Expect(err).NotTo(HaveOccurred())

		layer.Exec.Path = layer.Path
		layer.Metadata = map[string]interface{}{}
		layer.Profile = libcnb.Profile{}
	})

	it.After(func() {
		Expect(os.RemoveAll(layer.Path)).To(Succeed())
	})

	context("LayerContributor", func() {
		var (
			lc libpak.LayerContributor
		)

		it.Before(func() {
			lc.ExpectedMetadata = map[string]interface{}{
				"alpha": "test-alpha",
				"bravo": map[string]interface{}{
					"bravo-1": "test-bravo-1",
					"bravo-2": "test-bravo-2",
				},
			}
		})

		it("calls function with no existing metadata", func() {
			var called bool

			_, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "test-alpha"

			var called bool

			_, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("does not call function with matching metadata", func() {
			layer.Metadata = map[string]interface{}{
				"alpha": "test-alpha",
				"bravo": map[string]interface{}{
					"bravo-1": "test-bravo-1",
					"bravo-2": "test-bravo-2",
				},
			}

			var called bool

			_, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeFalse())
		})

		it("returns function error", func() {
			_, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				return libcnb.Layer{}, fmt.Errorf("test-error")
			})
			Expect(err).To(MatchError("test-error"))
		})

		it("adds expected metadata to layer", func() {
			layer, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"alpha": "test-alpha",
				"bravo": map[string]interface{}{
					"bravo-1": "test-bravo-1",
					"bravo-2": "test-bravo-2",
				},
			}))
		})

		it("sets build layer flag", func() {
			lc.ExpectedTypes.Build = true
			layer, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.LayerTypes.Build).To(BeTrue())
		})

		it("sets cache layer flag", func() {
			lc.ExpectedTypes.Cache = true
			layer, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.LayerTypes.Cache).To(BeTrue())
		})

		it("sets launch layer flag", func() {
			lc.ExpectedTypes.Launch = true
			layer, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.LayerTypes.Launch).To(BeTrue())
		})
	})

	context("NewDependencyLayer", func() {
		var dep libpak.BuildpackDependency

		it.Before(func() {
			dep = libpak.BuildpackDependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: "1.1.1",
				URI:     "test-uri",
				SHA256:  "576dd8416de5619ea001d9662291d62444d1292a38e96956bc4651c01f14bca1",
				Stacks:  []string{"test-stack"},
				Licenses: []libpak.BuildpackDependencyLicense{
					{
						Type: "test-type",
						URI:  "test-uri",
					},
				},
			}
		})
		it("returns a BOM entry for the layer", func() {
			_, entry := libpak.NewDependencyLayer(dep, libpak.DependencyCache{}, libcnb.LayerTypes{})
			Expect(entry.Name).To(Equal("test-id"))
			Expect(entry.Metadata["name"]).To(Equal("test-name"))
			Expect(entry.Metadata["version"]).To(Equal("1.1.1"))
			Expect(entry.Metadata["uri"]).To(Equal("test-uri"))
			Expect(entry.Metadata["sha256"]).To(Equal("576dd8416de5619ea001d9662291d62444d1292a38e96956bc4651c01f14bca1"))
			Expect(entry.Metadata["licenses"]).To(Equal([]libpak.BuildpackDependencyLicense{
				{
					Type: "test-type",
					URI:  "test-uri",
				},
			}))
		})
		context("launch layer type", func() {
			it("only sets launch on the entry", func() {
				_, entry := libpak.NewDependencyLayer(dep, libpak.DependencyCache{}, libcnb.LayerTypes{
					Launch: true,
				})
				Expect(entry.Launch).To(BeTrue())
				Expect(entry.Build).To(BeFalse())
			})
		})

		context("launch and build layer type", func() {
			it("sets launch and build on the entry", func() {
				_, entry := libpak.NewDependencyLayer(dep, libpak.DependencyCache{}, libcnb.LayerTypes{
					Launch: true,
					Build:  true,
				})
				Expect(entry.Launch).To(BeTrue())
				Expect(entry.Build).To(BeTrue())
			})
		})

		context("launch and cache layer type", func() {
			it("sets launch and build on the entry", func() {
				_, entry := libpak.NewDependencyLayer(dep, libpak.DependencyCache{}, libcnb.LayerTypes{
					Launch: true,
					Cache:  true,
				})
				Expect(entry.Launch).To(BeTrue())
				Expect(entry.Build).To(BeTrue())
			})
		})

		context("build layer type", func() {
			it("sets build on the entry", func() {
				_, entry := libpak.NewDependencyLayer(dep, libpak.DependencyCache{}, libcnb.LayerTypes{
					Build: true,
				})
				Expect(entry.Launch).To(BeFalse())
				Expect(entry.Build).To(BeTrue())
			})
		})

		context("cache layer type", func() {
			it("sets build on the entry", func() {
				_, entry := libpak.NewDependencyLayer(dep, libpak.DependencyCache{}, libcnb.LayerTypes{
					Cache: true,
				})
				Expect(entry.Launch).To(BeFalse())
				Expect(entry.Build).To(BeTrue())
			})
		})

		context("no layer types", func() {
			it("sets build on the entry", func() {
				_, entry := libpak.NewDependencyLayer(dep, libpak.DependencyCache{}, libcnb.LayerTypes{})
				Expect(entry.Launch).To(BeFalse())
				Expect(entry.Build).To(BeTrue())
			})
		})
	})

	context("DependencyLayerContributor", func() {
		var (
			dependency libpak.BuildpackDependency
			dlc        libpak.DependencyLayerContributor
			server     *ghttp.Server
		)

		it.Before(func() {
			RegisterTestingT(t)
			server = ghttp.NewServer()

			dependency = libpak.BuildpackDependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: "1.1.1",
				URI:     fmt.Sprintf("%s/test-path", server.URL()),
				SHA256:  "576dd8416de5619ea001d9662291d62444d1292a38e96956bc4651c01f14bca1",
				Stacks:  []string{"test-stack"},
				Licenses: []libpak.BuildpackDependencyLicense{
					{
						Type: "test-type",
						URI:  "test-uri",
					},
				},
			}

			layer.Metadata = map[string]interface{}{}

			dlc.ExpectedMetadata = dependency
			dlc.Dependency = dependency
			dlc.DependencyCache.CachePath = layer.Path
			dlc.DependencyCache.DownloadPath = layer.Path
		})

		it.After(func() {
			server.Close()
		})

		it("calls function with no existing metadata", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			var called bool

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("modifies request", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyHeaderKV("Test-Key", "test-value"),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			dlc.RequestModifierFuncs = append(dlc.RequestModifierFuncs, func(request *http.Request) (*http.Request, error) {
				request.Header.Add("Test-Key", "test-value")
				return request, nil
			})

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "test-alpha"

			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			var called bool

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("does not call function with matching metadata", func() {
			layer.Metadata = map[string]interface{}{
				"id":      dependency.ID,
				"name":    dependency.Name,
				"version": dependency.Version,
				"uri":     dependency.URI,
				"sha256":  dependency.SHA256,
				"stacks":  []interface{}{dependency.Stacks[0]},
				"licenses": []map[string]interface{}{
					{
						"type": dependency.Licenses[0].Type,
						"uri":  dependency.Licenses[0].URI,
					},
				},
			}

			var called bool

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeFalse())
		})

		it("returns function error", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				return libcnb.Layer{}, fmt.Errorf("test-error")
			})
			Expect(err).To(MatchError("test-error"))
		})

		it("adds expected metadata to layer", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			layer, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"id":      dependency.ID,
				"name":    dependency.Name,
				"version": dependency.Version,
				"uri":     dependency.URI,
				"sha256":  dependency.SHA256,
				"stacks":  []interface{}{dependency.Stacks[0]},
				"licenses": []map[string]interface{}{
					{
						"type": dependency.Licenses[0].Type,
						"uri":  dependency.Licenses[0].URI,
					},
				},
			}))
		})
	})

	context("NewHelperLayer", func() {
		it("returns a BOM entry with version equal to buildpack version", func() {
			_, entry := libpak.NewHelperLayer(libcnb.Buildpack{
				Info: libcnb.BuildpackInfo{
					Version: "test-version",
				},
			}, "test-name-1", "test-name-2")
			Expect(entry).To(Equal(
				libcnb.BOMEntry{
					Name: filepath.Base("helper"),
					Metadata: map[string]interface{}{
						"layer":   "helper",
						"names":   []string{"test-name-1", "test-name-2"},
						"version": "test-version",
					},
					Launch: true,
					Build:  false,
				},
			))
		})
	})

	context("HelperLayerContributor", func() {
		var (
			buildpack libcnb.Buildpack
			hlc       libpak.HelperLayerContributor
		)

		it.Before(func() {
			var err error

			buildpack.Info = libcnb.BuildpackInfo{
				ID:       "test-id",
				Name:     "test-name",
				Version:  "test-version",
				Homepage: "test-homepage",
			}

			buildpack.Path, err = ioutil.TempDir("", "buildpack")
			Expect(err).NotTo(HaveOccurred())

			file := filepath.Join(buildpack.Path, "bin")
			Expect(os.MkdirAll(file, 0755)).To(Succeed())

			file = filepath.Join(file, "helper")
			Expect(ioutil.WriteFile(file, []byte{}, 0755)).To(Succeed())

			hlc = libpak.HelperLayerContributor{
				Path:          file,
				BuildpackInfo: buildpack.Info,
				Logger:        bard.Logger{},
				Names:         []string{"test-name-1", "test-name-2"},
			}
		})

		it.After(func() {
			Expect(os.RemoveAll(buildpack.Path)).To(Succeed())
		})

		it("calls function with no existing metadata", func() {
			_, err := hlc.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(layer.Exec.FilePath("test-name-1"))).To(BeAnExistingFile())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "other-alpha"

			_, err := hlc.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			file := filepath.Join(layer.Exec.FilePath("test-name-1"))
			Expect(file).To(BeAnExistingFile())
			Expect(os.Readlink(file)).To(Equal(filepath.Join(layer.Path, "helper")))

			file = filepath.Join(layer.Exec.FilePath("test-name-2"))
			Expect(file).To(BeAnExistingFile())
			Expect(os.Readlink(file)).To(Equal(filepath.Join(layer.Path, "helper")))
		})

		it("does not call function with matching metadata", func() {
			layer.Metadata = map[string]interface{}{
				"id":        buildpack.Info.ID,
				"name":      buildpack.Info.Name,
				"version":   buildpack.Info.Version,
				"homepage":  buildpack.Info.Homepage,
				"clear-env": buildpack.Info.ClearEnvironment,
			}

			_, err := hlc.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(layer.Exec.FilePath("test-name-1"))).NotTo(BeAnExistingFile())
			Expect(filepath.Join(layer.Exec.FilePath("test-name-2"))).NotTo(BeAnExistingFile())
		})

		it("adds expected metadata to layer", func() {
			layer, err := hlc.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"id":        buildpack.Info.ID,
				"name":      buildpack.Info.Name,
				"version":   buildpack.Info.Version,
				"homepage":  buildpack.Info.Homepage,
				"clear-env": buildpack.Info.ClearEnvironment,
			}))
		})
	})
}
