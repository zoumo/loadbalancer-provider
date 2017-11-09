package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient=true
// +nonNamespaced=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageType describes the parameters for a class of storage for
// which PersistentVolumes can be dynamically provisioned.
//
// StorageTypes are non-namespaced; the name of the storage type
// according to etcd is in ObjectMeta.Name.
type StorageType struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Provisioner indicates the type of the provisioner.
	Provisioner string `json:"provisioner" protobuf:"bytes,2,opt,name=provisioner"`

	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage type.
	// Required ones for create storage service.
	// +optional
	RequiredParameters map[string]string `json:"requiredParameters,omitempty" protobuf:"bytes,3,rep,name=requiredParameters"`

	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage type.
	// Required ones for create storage class.
	// +optional
	OptionalParameters map[string]string `json:"optionalParameters,omitempty" protobuf:"bytes,3,rep,name=classOptionalParameters"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageTypeList is a collection of storage types.
type StorageTypeList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of StorageClasses
	Items []StorageType `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// StorageClassStatus is information about the current status of a StorageClass.
type StorageServiceStatus struct {
	// Phase is the current lifecycle phase of the storage class.
	// +optional
	Phase StorageServicePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=NamespacePhase"`
}

type StorageServicePhase string

// These are the valid phases of a storage service.
const (
	// StorageServiceActive means the storage class is available for use in the system
	StorageServiceActive StorageServicePhase = "Active"
	// StorageServiceTerminating means the storage class is undergoing graceful termination
	StorageServiceTerminating StorageServicePhase = "Terminating"
)

// +genclient=true
// +nonNamespaced=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageService describes the parameters for a class of storage for
// which PersistentVolumes can be dynamically provisioned.
//
// StorageServices are non-namespaced; the name of the storage service
// according to etcd is in ObjectMeta.Name.
type StorageService struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// TypeName indicates the name of the storage type that this service belongs to.
	TypeName string `json:"typeName" protobuf:"bytes,2,opt,name=typeName"`

	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage class.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`

	// Status describes the current status of a StorageService.
	// +optional
	Status StorageServiceStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageServiceList is a collection of storage services.
type StorageServiceList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of StorageClasses
	Items []StorageService `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// StorageClassStatus is information about the current status of a StorageClass.
type StorageClassStatus struct {
	// Phase is the current lifecycle phase of the storage class.
	// +optional
	Phase StorageClassPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=NamespacePhase"`
}

type StorageClassPhase string

// These are the valid phases of a storage class.
const (
	// StorageClassPending means the storage class is going to be available for use in the system
	StorageClassPending StorageClassPhase = "Pending"
	// StorageClassActive means the storage class is available for use in the system
	StorageClassActive StorageClassPhase = "Active"
	// StorageClassTerminating means the storage class is undergoing graceful termination
	StorageClassTerminating StorageClassPhase = "Terminating"
)

// +genclient=true
// +nonNamespaced=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageClass describes the parameters for a class of storage for
// which PersistentVolumes can be dynamically provisioned.
//
// StorageClass are non-namespaced; the name of the storage class
// according to etcd is in ObjectMeta.Name.
type StorageClass struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Provisioner indicates the type of the provisioner.
	Provisioner string `json:"provisioner" protobuf:"bytes,2,opt,name=provisioner"`

	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage class.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`

	// Status describes the current status of a StorageClass.
	// +optional
	Status StorageClassStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageClassList is a collection of storage classes.
type StorageClassList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is the list of StorageClasses
	Items []StorageClass `json:"items" protobuf:"bytes,2,rep,name=items"`
}
