// +build !ignore_autogenerated

/*
Copyright 2019 caicloud authors. All rights reserved.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvalFunc) DeepCopyInto(out *EvalFunc) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvalFunc.
func (in *EvalFunc) DeepCopy() *EvalFunc {
	if in == nil {
		return nil
	}
	out := new(EvalFunc)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvalJob) DeepCopyInto(out *EvalJob) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvalJob.
func (in *EvalJob) DeepCopy() *EvalJob {
	if in == nil {
		return nil
	}
	out := new(EvalJob)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EvalJob) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvalJobCondition) DeepCopyInto(out *EvalJobCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvalJobCondition.
func (in *EvalJobCondition) DeepCopy() *EvalJobCondition {
	if in == nil {
		return nil
	}
	out := new(EvalJobCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvalJobList) DeepCopyInto(out *EvalJobList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EvalJob, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvalJobList.
func (in *EvalJobList) DeepCopy() *EvalJobList {
	if in == nil {
		return nil
	}
	out := new(EvalJobList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EvalJobList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvalJobSpec) DeepCopyInto(out *EvalJobSpec) {
	*out = *in
	if in.CleanPodPolicy != nil {
		in, out := &in.CleanPodPolicy, &out.CleanPodPolicy
		*out = new(CleanPodPolicy)
		**out = **in
	}
	if in.Models != nil {
		in, out := &in.Models, &out.Models
		*out = make([]EvalModel, len(*in))
		copy(*out, *in)
	}
	if in.Functions != nil {
		in, out := &in.Functions, &out.Functions
		*out = make([]EvalFunc, len(*in))
		copy(*out, *in)
	}
	in.Template.DeepCopyInto(&out.Template)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvalJobSpec.
func (in *EvalJobSpec) DeepCopy() *EvalJobSpec {
	if in == nil {
		return nil
	}
	out := new(EvalJobSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvalJobStatus) DeepCopyInto(out *EvalJobStatus) {
	*out = *in
	in.WorkersStatus.DeepCopyInto(&out.WorkersStatus)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]EvalJobCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.StartTime != nil {
		in, out := &in.StartTime, &out.StartTime
		*out = (*in).DeepCopy()
	}
	if in.CompletionTime != nil {
		in, out := &in.CompletionTime, &out.CompletionTime
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvalJobStatus.
func (in *EvalJobStatus) DeepCopy() *EvalJobStatus {
	if in == nil {
		return nil
	}
	out := new(EvalJobStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvalModel) DeepCopyInto(out *EvalModel) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvalModel.
func (in *EvalModel) DeepCopy() *EvalModel {
	if in == nil {
		return nil
	}
	out := new(EvalModel)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EvalWorkersStatus) DeepCopyInto(out *EvalWorkersStatus) {
	*out = *in
	if in.Workers != nil {
		in, out := &in.Workers, &out.Workers
		*out = make(map[string]v1.PodPhase, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EvalWorkersStatus.
func (in *EvalWorkersStatus) DeepCopy() *EvalWorkersStatus {
	if in == nil {
		return nil
	}
	out := new(EvalWorkersStatus)
	in.DeepCopyInto(out)
	return out
}
