# Tagging
To help manage volumes in the aws account, CSI driver will automatically add tags to the volumes it manages.

| TagKey                 | TagValue                  | sample                                                              | Description         |
|------------------------|---------------------------|---------------------------------------------------------------------|---------------------|
| CSIVolumeName          | pvcName                   | CSIVolumeName = pvc-a3ab0567-3a48-4608-8cb6-4e3b1485c808            | add to all volumes, for recording associated pvc id and checking if a given volume was already created so that ControllerPublish/CreateVolume is idempotent. |
| CSIVolumeSnapshotName  | volumeSnapshotContentName | CSIVolumeSnapshotName = snapcontent-69477690-803b-4d3e-a61a-03c7b2592a76 | add to all snapshots, for recording associated VolumeSnapshot id and checking if a given snapshot was already created                                    |
| ebs.csi.aws.com/cluster| true                      | ebs.csi.aws.com/cluster = true                                      | add to all volumes and snapshots, for allowing users to use a policy to limit csi driver's permission to just the resources it manages.                      |
| kubernetes.io/cluster/X| owned                     | kubernetes.io/cluster/aws-cluster-id-1 = owned                      | add to all volumes and snapshots if k8s-tag-cluster-id argument is set to X.|
| extra-key              | extra-value               | extra-key = extra-value                                             | add to all volumes and snapshots if extraTags argument is set|

# StorageClass Tagging

The AWS EBS CSI Driver supports tagging through `StorageClass.parameters`. 

If a key has the prefix `tagSpecification`, the CSI driver will treat the value as a key-value pair to be applied to the dynamically provisioned volume as tags.


**Example 1**
```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: ebs-sc
provisioner: ebs.csi.aws.com
parameters:
  tagSpecification_1: "key1=value1"
  tagSpecification_2: "key2=hello world"
  tagSpecification_3: "key3="
```

Provisioning a volume using this StorageClass will apply two tags:

```
key1=value1
key2=hello world
key3=<empty string>
```

________

To allow for PV-level granularity, the CSI driver support runtime string interpolation on the tag values. You can specify placeholders for PVC namespace, PVC name and PV name, which will then be dynamically computed at runtime.

**NOTE: This requires the `--extra-create-metadata` flag to be enabled on the `external-provisioner` sidecar.**

**Example 2**
```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: ebs-sc
provisioner: ebs.csi.aws.com
parameters:
  tagSpecification_1: "pvcnamespace={{ .PVCNamespace }}"
  tagSpecification_2: "pvcname={{ .PVCName }}"
  tagSpecification_3: "pvname={{ .PVName }}"
```
Provisioning a volume using this StorageClass, with a PVC named 'ebs-claim' in namespace 'default', will apply the following tags:

```
pvcnamespace=default
pvcname=ebs-claim
pvname=<the computed pv name>
```


_________

The driver uses Go's `text/template` package for string interpolation. As such, cluster admins are free to use the constructs provided by the package (except for certain function, see `Failure Modes` below). To aid cluster admins to be more expressive, certain functions have been provided.

They include:

-   **field** delim index str: Split `str` by `delim` and extract the  word at position `index`.
-   **substring** start end str: Get a substring of `str` given the `start` and `end` indices
-   **toUpper** str: Convert `str` to uppercase
-   **toLower** str: Convert `str` to lowercase
-   **contains** str1 str2: Returns a boolean if `str2` contains `str1`


**Example 3**
```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: ebs-sc
provisioner: ebs.csi.aws.com
parameters:
  tagSpecification_1: 'backup={{ .PVCNamespace | contains "prod" }}'
  tagSpecification_2: 'billingID={{ .PVCNamespace | field "-" 2 | toUpper }}'
```

Assuming the PVC namespace is `ns-prod-abcdef`, the attached tags will be

```
backup=true
billingID=ABCDEF
```

# Adding, Modifying, and Deleting Tags Of Existing Volumes
The AWS EBS CSI Driver supports the modifying of tags of existing volumes through `VolumeAttributesClass.parameters` the examples below show the syntax for addition, modification, and deletion of tags within the `VolumeAttributesClass.parameters`. The driver also supports runtime string interpolation on tag values for a volume upon modification, which allows the specification of placeholder values for the PVC namespace, PVC name, and PV name, which will then be dynamically computed at runtime. 

**Note: Interpolated tags require the `--extra-modify-metadata` flag to be enabled on the `external-resizer` sidecar. To modify Amazon EBS resource tags through VACs, ensure that you attach the following IAM Policy to the role used by your Amazon EBS CSI driver:** 
``` 
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateTags"
      ],
      "Resource": [
        "arn:aws:ec2:*:*:volume/*",
        "arn:aws:ec2:*:*:snapshot/*"
      ]
    }
  ]
}
```
**Syntax for Adding or Modifying a Tag**

If a key has the prefix `tagSpecification`, the CSI driver will treat the value as a key-value pair to be added to the existing volume. If there is already an existing tag with the specified key, the CSI driver will overwrite the value of that tag with the new value specified. 
```
apiVersion: storage.k8s.io/v1beta1
kind: VolumeAttributesClass
metadata:
  name: io2-class
driverName: ebs.csi.aws.com
parameters:
  tagSpecification_1: "location=Seattle"
  tagSpecification_2: "cost-center=" // If the value is left blank, tag is created with an empty value
  # Interpolated tag
  tagSpecification_3: "PVC-Name={{ .PVCName }}"
  tagSpecification_4: "PVC-Namespace={{ .PVCNamespace }}"
  tagSpecification_5: "PV-Name={{ .PVName }}"
  # Interpolated tag w/ function
  tagSpecification_6: 'key6={{ .PVCNamespace | contains "prod" }}'
```
**Syntax for Deleting a Tag**

If a key has the prefix `tagDeletion`, the CSI driver will treat the value as a tag key, and the existing tag with that key will be removed from the volume.
```
apiVersion: storage.k8s.io/v1beta1
kind: VolumeAttributesClass
metadata:
  name: io2-class
driverName: ebs.csi.aws.com
parameters:
  tagDeletion_1: "location" // Deletes tag with key "location"
  tagDeletion_2: "cost-center"
```

# Snapshot Tagging
The AWS EBS CSI Driver supports tagging snapshots through `VolumeSnapshotClass.parameters`, similarly to StorageClass tagging.

The CSI driver supports runtime string interpolation on the snapshot tag values. You can specify placeholders for VolumeSnapshot namespace, VolumeSnapshot name and VolumeSnapshotContent name, which will then be dynamically computed at runtime. You can also use the functions provided by the CSI Driver to apply more expressive tags. **Note: Interpolated tags require the `--extra-create-metadata` flag to be enabled on the `external-snapshotter` sidecar.**

**Example**
```
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshotClass
metadata:
  name: csi-aws-vsc
driver: ebs.csi.aws.com
deletionPolicy: Delete
parameters:
  tagSpecification_1: "key1=value1"
  tagSpecification_2: "key2="
  # Interpolated tag
  tagSpecification_3: "snapshotnamespace={{ .VolumeSnapshotNamespace }}"
  tagSpecification_4: "snapshotname={{ .VolumeSnapshotName }}"
  tagSpecification_5: "snapshotcontentname={{ .VolumeSnapshotContentName }}"
  # Interpolated tag w/ function
  tagSpecification_6: 'key6={{ .VolumeSnapshotNamespace | contains "prod" }}'
```

Provisioning a snapshot in namespace 'ns-prod' with `VolumeSnapshot` name being 'ebs-snapshot' using this VolumeSnapshotClass, will apply the following tags to the snapshot:

```
key1=value1
key2=<empty string>
snapshotnamespace=ns-prod
snapshotname=ebs-snapshot
snapshotcontentname=<the computed VolumeSnapshotContent name>
key6=true
```
____

## Failure Modes

There can be multipe failure modes:

* The template cannot be parsed.
* The key/interpolated value do not meet the [AWS Tag Requirements](https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html)
* The key is not allowed (such as keys used internally by the CSI driver e.g., 'CSIVolumeName').
* The template uses one of the disabled function calls. The driver disables the following `text/template` functions: `js`, `call`, `html`, `urlquery`. 

In this case, the CSI driver will not provision a volume, but instead return an error.

The driver also defines another flag, `--warn-on-invalid-tag` that will (if set), instead of returning an error, log a warning and skip the offending tag.


