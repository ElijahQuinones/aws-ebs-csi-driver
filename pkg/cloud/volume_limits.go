// Copyright 2024 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloud

import (
	"regexp"
	"strings"
)

// / https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html#ec2-nitro-instances
const (
	highMemoryMetalInstancesMaxVolumes   = 19
	highMemoryVirtualInstancesMaxVolumes = 27
	baremetalMaxVolumes                  = 31
	nonNitroMaxAttachments               = 39
	nitroMaxAttachments                  = 28
)

func init() {
	// This list of Nitro instance types have a dedicated Amazon EBS volume limit of up to 128 attachments, depending on instance size.
	// The limit is not shared with other device attachments: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/volume_limits.html#nitro-system-limits
	instanceFamilies := []string{"m7i", "m7i-flex", "m7a", "c7i", "c7i-flex", "c7a", "r7a", "r7i", "r7iz", "r8g", "u7i", "g6", "gr6"}
	commonInstanceSizes := []string{"medium", "large", "xlarge", "2xlarge", "4xlarge", "8xlarge", "12xlarge"}

	for _, family := range instanceFamilies {
		for _, size := range commonInstanceSizes {
			dedicatedVolumeLimits[family+"."+size] = 32
		}
		dedicatedVolumeLimits[family+".metal-16xl"] = 39
		dedicatedVolumeLimits[family+".metal-24xl"] = 39
		dedicatedVolumeLimits[family+".16xlarge"] = 48
		dedicatedVolumeLimits[family+".24xlarge"] = 64
		dedicatedVolumeLimits[family+".metal-32xl"] = 79
		dedicatedVolumeLimits[family+".metal-48xl"] = 79
		dedicatedVolumeLimits[family+".32xlarge"] = 88
		dedicatedVolumeLimits[family+".48xlarge"] = 128
	}
}

var dedicatedVolumeLimits = map[string]int{}

// / List of nitro instance types can be found here: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html#ec2-nitro-instances
var nonNitroInstanceFamilies = map[string]struct{}{
	"t2":  {},
	"c3":  {},
	"m3":  {},
	"r3":  {},
	"c4":  {},
	"m4":  {},
	"r4":  {},
	"x1e": {},
	"x1":  {},
	"p2":  {},
	"p3":  {},
	"g3":  {},
	"d2":  {},
	"h1":  {},
}

func IsNitroInstanceType(it string) bool {
	strs := strings.Split(it, ".")

	if len(strs) != 2 {
		panic("cannot determine family of instance type")
	}

	family := strs[0]
	_, ok := nonNitroInstanceFamilies[family]
	return !ok
}

func GetMaxAttachments(nitro bool) int {
	if nitro {
		return nitroMaxAttachments
	}
	return nonNitroMaxAttachments
}

// / Some instance types have a maximum limit of EBS volumes
// / https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/volume_limits.html
var maxVolumeLimits = map[string]int{
	"d3.8xlarge":    3,
	"d3en.12xlarge": 3,
	"g5.48xlarge":   9,
	"inf1.xlarge":   26,
	"inf1.2xlarge":  26,
	"inf1.6xlarge":  23,
	"inf1.24xlarge": 11,
	"mac1.metal":    16,
}

func GetEBSLimitForInstanceType(it string) (int, bool) {
	if v, ok := maxVolumeLimits[it]; ok {
		return v, ok
	}

	highMemoryMetalRegex := `^u-[a-z0-9]+\.metal$`
	re := regexp.MustCompile(highMemoryMetalRegex)

	if ok := re.MatchString(it); ok {
		return highMemoryMetalInstancesMaxVolumes, true
	}

	highMemoryVirtualRegex := `^u-[a-z0-9]+\.[a-z0-9]+`
	re = regexp.MustCompile(highMemoryVirtualRegex)

	if ok := re.MatchString(it); ok {
		return highMemoryVirtualInstancesMaxVolumes, true
	}

	bareMetalRegex := `[a-z0-9]+\.metal$`
	re = regexp.MustCompile(bareMetalRegex)

	if ok := re.MatchString(it); ok {
		return baremetalMaxVolumes, true
	}

	return 0, false
}

func GetDedicatedLimitForInstanceType(it string) int {
	if limit, ok := dedicatedVolumeLimits[it]; ok {
		return limit
	} else {
		return 0
	}
}

// GetReservedSlotsForInstanceType calculates how many attachment slots are already used up by other devices on shared EBS volume limit instances.
func GetReservedSlotsForInstanceType(it string) int {
	total := 0
	nvmeInstanceStoreVolumes, ok := nvmeInstanceStoreVolumes[it]
	if ok {
		total += nvmeInstanceStoreVolumes
	}
	gpus, ok := gpuInstanceGpus[it]
	if ok {
		total += gpus
	}
	return total
}

// / https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-store-volumes.html
// / IMDS does not provide NVMe instance store data; we'll just list all instances here
// / TODO: See if we can get these values from DescribeInstanceTypes API
var nvmeInstanceStoreVolumes = map[string]int{
	"c1.medium":       1,
	"c1.xlarge":       4,
	"c3.2xlarge":      2,
	"c3.4xlarge":      2,
	"c3.8xlarge":      2,
	"c3.large":        2,
	"c3.xlarge":       2,
	"c5ad.12xlarge":   2,
	"c5ad.16xlarge":   2,
	"c5ad.24xlarge":   2,
	"c5ad.2xlarge":    1,
	"c5ad.4xlarge":    2,
	"c5ad.8xlarge":    2,
	"c5ad.large":      1,
	"c5ad.xlarge":     1,
	"c5d.12xlarge":    2,
	"c5d.18xlarge":    2,
	"c5d.24xlarge":    4,
	"c5d.2xlarge":     1,
	"c5d.4xlarge":     1,
	"c5d.9xlarge":     1,
	"c5d.large":       1,
	"c5d.metal":       4,
	"c5d.xlarge":      1,
	"c6gd.12xlarge":   2,
	"c6gd.16xlarge":   2,
	"c6gd.2xlarge":    1,
	"c6gd.4xlarge":    1,
	"c6gd.8xlarge":    1,
	"c6gd.large":      1,
	"c6gd.medium":     1,
	"c6gd.metal":      2,
	"c6gd.xlarge":     1,
	"c6id.12xlarge":   2,
	"c6id.16xlarge":   2,
	"c6id.24xlarge":   4,
	"c6id.2xlarge":    1,
	"c6id.32xlarge":   4,
	"c6id.4xlarge":    1,
	"c6id.8xlarge":    1,
	"c6id.large":      1,
	"c6id.metal":      4,
	"c6id.xlarge":     1,
	"c7gd.12xlarge":   2,
	"c7gd.16xlarge":   2,
	"c7gd.2xlarge":    1,
	"c7gd.4xlarge":    1,
	"c7gd.8xlarge":    1,
	"c7gd.large":      1,
	"c7gd.medium":     1,
	"c7gd.metal":      2,
	"c7gd.xlarge":     1,
	"d2.2xlarge":      6,
	"d2.4xlarge":      12,
	"d2.8xlarge":      24,
	"d2.xlarge":       3,
	"d3.2xlarge":      6,
	"d3.4xlarge":      12,
	"d3.8xlarge":      24,
	"d3en.12xlarge":   24,
	"d3en.2xlarge":    4,
	"d3en.4xlarge":    8,
	"d3en.6xlarge":    12,
	"d3en.8xlarge":    16,
	"d3en.xlarge":     2,
	"d3.xlarge":       3,
	"dl1.24xlarge":    4,
	"f1.16xlarge":     4,
	"f1.2xlarge":      1,
	"f1.4xlarge":      1,
	"g4ad.16xlarge":   2,
	"g4ad.2xlarge":    1,
	"g4ad.4xlarge":    1,
	"g4ad.8xlarge":    1,
	"g4ad.xlarge":     1,
	"g4dn.12xlarge":   1,
	"g4dn.16xlarge":   1,
	"g4dn.2xlarge":    1,
	"g4dn.4xlarge":    1,
	"g4dn.8xlarge":    1,
	"g4dn.metal":      2,
	"g4dn.xlarge":     1,
	"g5.12xlarge":     1,
	"g5.16xlarge":     1,
	"g5.24xlarge":     1,
	"g5.2xlarge":      1,
	"g5.48xlarge":     2,
	"g5.4xlarge":      1,
	"g5.8xlarge":      1,
	"g5.xlarge":       1,
	"g6.12xlarge":     4,
	"g6.16xlarge":     2,
	"g6.24xlarge":     4,
	"g6.2xlarge":      1,
	"g6.48xlarge":     8,
	"g6.4xlarge":      1,
	"g6.8xlarge":      2,
	"g6.xlarge":       1,
	"gr6.4xlarge":     1,
	"gr6.8xlarge":     2,
	"h1.16xlarge":     8,
	"h1.2xlarge":      1,
	"h1.4xlarge":      2,
	"h1.8xlarge":      4,
	"hpc6id.32xlarge": 4,
	"i2.2xlarge":      2,
	"i2.4xlarge":      4,
	"i2.8xlarge":      8,
	"i2.xlarge":       1,
	"i3.16xlarge":     8,
	"i3.2xlarge":      1,
	"i3.4xlarge":      2,
	"i3.8xlarge":      4,
	"i3en.12xlarge":   4,
	"i3en.24xlarge":   8,
	"i3en.2xlarge":    2,
	"i3en.3xlarge":    1,
	"i3en.6xlarge":    2,
	"i3en.large":      1,
	"i3en.metal":      8,
	"i3en.xlarge":     1,
	"i3.large":        1,
	"i3.metal":        8,
	"i3.xlarge":       1,
	"i4g.16xlarge":    4,
	"i4g.2xlarge":     1,
	"i4g.4xlarge":     1,
	"i4g.8xlarge":     2,
	"i4g.large":       1,
	"i4g.xlarge":      1,
	"i4i.12xlarge":    3,
	"i4i.16xlarge":    4,
	"i4i.24xlarge":    6,
	"i4i.2xlarge":     1,
	"i4i.32xlarge":    8,
	"i4i.4xlarge":     1,
	"i4i.8xlarge":     2,
	"i4i.large":       1,
	"i4i.metal":       8,
	"i4i.xlarge":      1,
	"im4gn.16xlarge":  4,
	"im4gn.2xlarge":   1,
	"im4gn.4xlarge":   1,
	"im4gn.8xlarge":   2,
	"im4gn.large":     1,
	"im4gn.xlarge":    1,
	"is4gen.2xlarge":  1,
	"is4gen.4xlarge":  2,
	"is4gen.8xlarge":  4,
	"is4gen.large":    1,
	"is4gen.medium":   1,
	"is4gen.xlarge":   1,
	"m1.large":        2,
	"m1.medium":       1,
	"m1.small":        1,
	"m1.xlarge":       4,
	"m2.2xlarge":      1,
	"m2.4xlarge":      2,
	"m2.xlarge":       1,
	"m3.2xlarge":      2,
	"m3.large":        1,
	"m3.medium":       1,
	"m3.xlarge":       2,
	"m5ad.12xlarge":   2,
	"m5ad.16xlarge":   4,
	"m5ad.24xlarge":   4,
	"m5ad.2xlarge":    1,
	"m5ad.4xlarge":    2,
	"m5ad.8xlarge":    2,
	"m5ad.large":      1,
	"m5ad.xlarge":     1,
	"m5d.12xlarge":    2,
	"m5d.16xlarge":    4,
	"m5d.24xlarge":    4,
	"m5d.2xlarge":     1,
	"m5d.4xlarge":     2,
	"m5d.8xlarge":     2,
	"m5d.large":       1,
	"m5d.metal":       4,
	"m5dn.12xlarge":   2,
	"m5dn.16xlarge":   4,
	"m5dn.24xlarge":   4,
	"m5dn.2xlarge":    1,
	"m5dn.4xlarge":    2,
	"m5dn.8xlarge":    2,
	"m5dn.large":      1,
	"m5dn.metal":      4,
	"m5dn.xlarge":     1,
	"m5d.xlarge":      1,
	"m6gd.12xlarge":   2,
	"m6gd.16xlarge":   2,
	"m6gd.2xlarge":    1,
	"m6gd.4xlarge":    1,
	"m6gd.8xlarge":    1,
	"m6gd.large":      1,
	"m6gd.medium":     1,
	"m6gd.metal":      2,
	"m6gd.xlarge":     1,
	"m6id.12xlarge":   2,
	"m6id.16xlarge":   2,
	"m6id.24xlarge":   4,
	"m6id.2xlarge":    1,
	"m6id.32xlarge":   4,
	"m6id.4xlarge":    1,
	"m6id.8xlarge":    1,
	"m6id.large":      1,
	"m6id.metal":      4,
	"m6idn.12xlarge":  2,
	"m6idn.16xlarge":  2,
	"m6idn.24xlarge":  4,
	"m6idn.2xlarge":   1,
	"m6idn.32xlarge":  4,
	"m6idn.4xlarge":   1,
	"m6idn.8xlarge":   1,
	"m6idn.large":     1,
	"m6idn.metal":     4,
	"m6idn.xlarge":    1,
	"m6id.xlarge":     1,
	"m7gd.12xlarge":   2,
	"m7gd.16xlarge":   2,
	"m7gd.2xlarge":    1,
	"m7gd.4xlarge":    1,
	"m7gd.8xlarge":    1,
	"m7gd.large":      1,
	"m7gd.medium":     1,
	"m7gd.metal":      2,
	"m7gd.xlarge":     1,
	"p3dn.24xlarge":   2,
	"p4d.24xlarge":    8,
	"p4de.24xlarge":   8,
	"p5.48xlarge":     8,
	"r3.2xlarge":      1,
	"r3.4xlarge":      1,
	"r3.8xlarge":      2,
	"r3.large":        1,
	"r3.xlarge":       1,
	"r5ad.12xlarge":   2,
	"r5ad.16xlarge":   4,
	"r5ad.24xlarge":   4,
	"r5ad.2xlarge":    1,
	"r5ad.4xlarge":    2,
	"r5ad.8xlarge":    2,
	"r5ad.large":      1,
	"r5ad.xlarge":     1,
	"r5d.12xlarge":    2,
	"r5d.16xlarge":    4,
	"r5d.24xlarge":    4,
	"r5d.2xlarge":     1,
	"r5d.4xlarge":     2,
	"r5d.8xlarge":     2,
	"r5d.large":       1,
	"r5d.metal":       4,
	"r5dn.12xlarge":   2,
	"r5dn.16xlarge":   4,
	"r5dn.24xlarge":   4,
	"r5dn.2xlarge":    1,
	"r5dn.4xlarge":    2,
	"r5dn.8xlarge":    2,
	"r5dn.large":      1,
	"r5dn.metal":      4,
	"r5dn.xlarge":     1,
	"r5d.xlarge":      1,
	"r6gd.12xlarge":   2,
	"r6gd.16xlarge":   2,
	"r6gd.2xlarge":    1,
	"r6gd.4xlarge":    1,
	"r6gd.8xlarge":    1,
	"r6gd.large":      1,
	"r6gd.medium":     1,
	"r6gd.metal":      2,
	"r6gd.xlarge":     1,
	"r6id.12xlarge":   2,
	"r6id.16xlarge":   2,
	"r6id.24xlarge":   4,
	"r6id.2xlarge":    1,
	"r6id.32xlarge":   4,
	"r6id.4xlarge":    1,
	"r6id.8xlarge":    1,
	"r6id.large":      1,
	"r6id.metal":      4,
	"r6idn.12xlarge":  2,
	"r6idn.16xlarge":  2,
	"r6idn.24xlarge":  4,
	"r6idn.2xlarge":   1,
	"r6idn.32xlarge":  4,
	"r6idn.4xlarge":   1,
	"r6idn.8xlarge":   1,
	"r6idn.large":     1,
	"r6idn.metal":     4,
	"r6idn.xlarge":    1,
	"r6id.xlarge":     1,
	"r7gd.12xlarge":   2,
	"r7gd.16xlarge":   2,
	"r7gd.2xlarge":    1,
	"r7gd.4xlarge":    1,
	"r7gd.8xlarge":    1,
	"r7gd.large":      1,
	"r7gd.medium":     1,
	"r7gd.metal":      2,
	"r7gd.xlarge":     1,
	"trn1.2xlarge":    1,
	"trn1.32xlarge":   4,
	"trn1n.32xlarge":  4,
	"x1.16xlarge":     1,
	"x1.32xlarge":     2,
	"x1e.16xlarge":    1,
	"x1e.2xlarge":     1,
	"x1e.32xlarge":    2,
	"x1e.4xlarge":     1,
	"x1e.8xlarge":     1,
	"x1e.xlarge":      1,
	"x2gd.12xlarge":   2,
	"x2gd.16xlarge":   2,
	"x2gd.2xlarge":    1,
	"x2gd.4xlarge":    1,
	"x2gd.8xlarge":    1,
	"x2gd.large":      1,
	"x2gd.medium":     1,
	"x2gd.metal":      2,
	"x2gd.xlarge":     1,
	"x2idn.16xlarge":  1,
	"x2idn.24xlarge":  2,
	"x2idn.32xlarge":  2,
	"x2idn.metal":     2,
	"x2iedn.16xlarge": 1,
	"x2iedn.24xlarge": 2,
	"x2iedn.2xlarge":  1,
	"x2iedn.32xlarge": 2,
	"x2iedn.4xlarge":  1,
	"x2iedn.8xlarge":  1,
	"x2iedn.metal":    2,
	"x2iedn.xlarge":   1,
	"z1d.12xlarge":    2,
	"z1d.2xlarge":     1,
	"z1d.3xlarge":     1,
	"z1d.6xlarge":     1,
	"z1d.large":       1,
	"z1d.metal":       2,
	"z1d.xlarge":      1,
}

// / https://aws.amazon.com/ec2/instance-types
var gpuInstanceGpus = map[string]int{
	"dl1.24xlarge":  8,
	"g3.16xlarge":   4,
	"g3.4xlarge":    1,
	"g3.8xlarge":    2,
	"g3s.xlarge":    1,
	"g4ad.16xlarge": 4,
	"g4ad.2xlarge":  1,
	"g4ad.4xlarge":  1,
	"g4ad.8xlarge":  2,
	"g4ad.xlarge":   1,
	"g4dn.12xlarge": 4,
	"g4dn.16xlarge": 1,
	"g4dn.2xlarge":  1,
	"g4dn.4xlarge":  1,
	"g4dn.8xlarge":  1,
	"g4dn.metal":    8,
	"g4dn.xlarge":   1,
	"g5.12xlarge":   4,
	"g5.16xlarge":   1,
	"g5.24xlarge":   4,
	"g5.2xlarge":    1,
	"g5.48xlarge":   8,
	"g5.4xlarge":    1,
	"g5.8xlarge":    1,
	"g5g.16xlarge":  2,
	"g5g.2xlarge":   1,
	"g5g.4xlarge":   1,
	"g5g.8xlarge":   1,
	"g5g.metal":     2,
	"g5g.xlarge":    1,
	"g5.xlarge":     1,
	"g6.12xlarge":   4,
	"g6.16xlarge":   1,
	"g6.24xlarge":   4,
	"g6.2xlarge":    1,
	"g6.48xlarge":   8,
	"g6.4xlarge":    1,
	"g6.8xlarge":    1,
	"g6.xlarge":     1,
	"gr6.4xlarge":   1,
	"gr6.8xlarge":   1,
	"p2.16xlarge":   16,
	"p2.8xlarge":    8,
	"p2.xlarge":     1,
	"p3.16xlarge":   8,
	"p3.2xlarge":    1,
	"p3.8xlarge":    4,
	"p3dn.24xlarge": 8,
	"p4d.24xlarge":  8,
	"p4de.24xlarge": 8,
	"p5.48xlarge":   8,
}
