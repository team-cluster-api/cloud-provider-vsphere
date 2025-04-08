/*
Copyright 2021 The Kubernetes Authors.

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

package vmservice

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/klogr"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmop "k8s.io/cloud-provider-vsphere/pkg/cloudprovider/vsphereparavirtual/vmoperator"
)

var log = klogr.New().WithName("vmservice")

// VMService is an interface for VirtualMachineService operations
type VMService interface {
	GetVMServiceName(service *v1.Service, clusterName string) string
	Get(ctx context.Context, service *v1.Service, clusterName string) (*vmopv1.VirtualMachineService, error)
	Create(ctx context.Context, service *v1.Service, clusterName string) (*vmopv1.VirtualMachineService, error)
	CreateOrUpdate(ctx context.Context, service *v1.Service, clusterName string) (*vmopv1.VirtualMachineService, error)
	Update(ctx context.Context, service *v1.Service, clusterName string, vmService *vmopv1.VirtualMachineService) (*vmopv1.VirtualMachineService, error)
	Delete(ctx context.Context, service *v1.Service, clusterName string) error
}

// vmService takes care of mapping of LB type of service to VM service in supervisor cluster
type vmService struct {
	vmClient                            vmop.Interface
	namespace                           string
	ownerReference                      *metav1.OwnerReference
	serviceAnnotationPropagationEnabled bool
}
