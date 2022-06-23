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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("test iterative digits count", func() {
	for i := 0; i < 10; i++ {
		Expect(IterativeDigitsCount(i)).To(Equal(1))
	}

	for i := 10; i < 100; i++ {
		Expect(IterativeDigitsCount(i)).To(Equal(2))
	}

	for i := 100; i < 1000; i++ {
		Expect(IterativeDigitsCount(i)).To(Equal(3))
	}
})
