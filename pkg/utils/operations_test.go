/*
Copyright The CloudNativePG Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	config "github.com/EnterpriseDB/cloud-native-postgresql/internal/configuration"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Difference of values of maps", func() {
	p1 := make(map[string]string, 2)
	const testString = "test"
	p1["t"] = testString
	p1["r"] = "rest"
	It("is nil when maps contains same key/value pairs", func() {
		p2 := make(map[string]string, 2)
		p2["t"] = testString
		p2["r"] = "rest"
		Expect(CollectDifferencesFromMaps(p1, p2)).To(BeNil())
	})

	It("is a list of string with difference when maps contains different key/value pairs", func() {
		p2 := make(map[string]string, 2)
		p2["t"] = testString
		p2["a"] = "apple"
		res := CollectDifferencesFromMaps(p1, p2)
		Expect(len(res)).To(BeEquivalentTo(2))
	})
})

var _ = Describe("Testing Annotations and labels subset", func() {
	const environment = "environment"
	const department = "finance"
	subSet := map[string]string{
		environment: "test",
		department:  "finance",
	}
	set := map[string]string{
		environment:   "test",
		"application": "game-history",
	}

	It("should make sure that a contained annotations subset is recognized", func() {
		isSubset := IsAnnotationSubset(set, subSet, &config.Data{
			InheritedAnnotations: []string{environment},
		})
		Expect(isSubset).To(BeTrue())
	})

	It("should make sure that a annotations non-subset is recognized", func() {
		isSubset := IsAnnotationSubset(set, subSet, &config.Data{
			InheritedAnnotations: []string{environment, department},
		})
		Expect(isSubset).To(BeFalse())
	})

	It("should make sure that a contained labels subset is recognized", func() {
		isSubset := IsLabelSubset(set, subSet, &config.Data{
			InheritedLabels: []string{environment},
		})
		Expect(isSubset).To(BeTrue())
	})

	It("should make sure that a labels non-subset is recognized", func() {
		isSubset := IsLabelSubset(set, subSet, &config.Data{
			InheritedLabels: []string{environment, department},
		})
		Expect(isSubset).To(BeFalse())
	})
})
