/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// FunctionValue
type FunctionValue interface {
	Value
	isFunctionValue()
	FunctionType() *sema.FunctionType
	// invoke evaluates the function.
	// Only used internally by the interpreter.
	// Use Interpreter.InvokeFunctionValue if you want to invoke the function externally
	invoke(Invocation) Value
}

// InterpretedFunctionValue
type InterpretedFunctionValue struct {
	Interpreter      *Interpreter
	ParameterList    *ast.ParameterList
	Type             *sema.FunctionType
	Activation       *VariableActivation
	BeforeStatements []ast.Statement
	PreConditions    ast.Conditions
	Statements       []ast.Statement
	PostConditions   ast.Conditions
}

func NewInterpretedFunctionValue(
	interpreter *Interpreter,
	parameterList *ast.ParameterList,
	functionType *sema.FunctionType,
	lexicalScope *VariableActivation,
	beforeStatements []ast.Statement,
	preConditions ast.Conditions,
	statements []ast.Statement,
	postConditions ast.Conditions,
) *InterpretedFunctionValue {

	common.UseMemory(interpreter, common.InterpretedFunctionValueMemoryUsage)

	return &InterpretedFunctionValue{
		Interpreter:      interpreter,
		ParameterList:    parameterList,
		Type:             functionType,
		Activation:       lexicalScope,
		BeforeStatements: beforeStatements,
		PreConditions:    preConditions,
		Statements:       statements,
		PostConditions:   postConditions,
	}
}

var _ Value = &InterpretedFunctionValue{}
var _ FunctionValue = &InterpretedFunctionValue{}

func (*InterpretedFunctionValue) IsValue() {}

func (f *InterpretedFunctionValue) String() string {
	return format.Function(f.Type.String())
}

func (f *InterpretedFunctionValue) RecursiveString(_ SeenReferences) string {
	return f.String()
}

func (f *InterpretedFunctionValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	// TODO: Meter sema.Type String conversion
	typeString := f.Type.String()
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(8+len(typeString)))
	return f.String()
}

func (f *InterpretedFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInterpretedFunctionValue(interpreter, f)
}

func (f *InterpretedFunctionValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (f *InterpretedFunctionValue) StaticType(interpreter *Interpreter) StaticType {
	return ConvertSemaToStaticType(interpreter, f.Type)
}

func (*InterpretedFunctionValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (*InterpretedFunctionValue) isFunctionValue() {}

func (f *InterpretedFunctionValue) FunctionType() *sema.FunctionType {
	return f.Type
}

func (f *InterpretedFunctionValue) invoke(invocation Invocation) Value {

	// The check that arguments' dynamic types match the parameter types
	// was already performed by the interpreter's checkValueTransferTargetType function

	return f.Interpreter.invokeInterpretedFunction(f, invocation)
}

func (f *InterpretedFunctionValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (f *InterpretedFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (*InterpretedFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*InterpretedFunctionValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (f *InterpretedFunctionValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return f
}

func (f *InterpretedFunctionValue) Clone(_ *Interpreter) Value {
	return f
}

func (*InterpretedFunctionValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

// HostFunctionValue
type HostFunction func(invocation Invocation) Value

type HostFunctionValue struct {
	Function        HostFunction
	NestedVariables map[string]*Variable
	Type            *sema.FunctionType
}

func (f *HostFunctionValue) String() string {
	return format.Function(f.Type.String())
}

func (f *HostFunctionValue) RecursiveString(_ SeenReferences) string {
	return f.String()
}

func (f *HostFunctionValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(memoryGauge, common.HostFunctionValueStringMemoryUsage)
	return f.String()
}

func NewUnmeteredHostFunctionValue(
	function HostFunction,
	funcType *sema.FunctionType,
) *HostFunctionValue {
	// Host functions can be passed by value,
	// so for the interpreter value transfer check to work,
	// they need a static type
	if funcType == nil {
		panic(errors.NewUnreachableError())
	}

	return &HostFunctionValue{
		Function: function,
		Type:     funcType,
	}
}

func NewHostFunctionValue(
	gauge common.MemoryGauge,
	function HostFunction,
	funcType *sema.FunctionType,
) *HostFunctionValue {

	common.UseMemory(gauge, common.HostFunctionValueMemoryUsage)

	return NewUnmeteredHostFunctionValue(function, funcType)
}

var _ Value = &HostFunctionValue{}
var _ FunctionValue = &HostFunctionValue{}
var _ MemberAccessibleValue = &HostFunctionValue{}
var _ ContractValue = &HostFunctionValue{}

func (*HostFunctionValue) IsValue() {}

func (f *HostFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitHostFunctionValue(interpreter, f)
}

func (f *HostFunctionValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (f *HostFunctionValue) StaticType(interpreter *Interpreter) StaticType {
	return ConvertSemaToStaticType(interpreter, f.Type)
}

func (*HostFunctionValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (*HostFunctionValue) isFunctionValue() {}

func (f *HostFunctionValue) FunctionType() *sema.FunctionType {
	return f.Type
}

func (f *HostFunctionValue) invoke(invocation Invocation) Value {

	// The check that arguments' dynamic types match the parameter types
	// was already performed by the interpreter's checkValueTransferTargetType function

	return f.Function(invocation)
}

func (f *HostFunctionValue) GetMember(_ *Interpreter, _ LocationRange, name string) Value {
	if f.NestedVariables != nil {
		if variable, ok := f.NestedVariables[name]; ok {
			return variable.GetValue()
		}
	}
	return nil
}

func (*HostFunctionValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Host functions have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*HostFunctionValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) {
	// Host functions have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (f *HostFunctionValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (f *HostFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (*HostFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*HostFunctionValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (f *HostFunctionValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return f
}

func (f *HostFunctionValue) Clone(_ *Interpreter) Value {
	return f
}

func (*HostFunctionValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v *HostFunctionValue) SetNestedVariables(variables map[string]*Variable) {
	v.NestedVariables = variables
}

// BoundFunctionValue
type BoundFunctionValue struct {
	Function FunctionValue
	Self     *MemberAccessibleValue
}

var _ Value = BoundFunctionValue{}
var _ FunctionValue = BoundFunctionValue{}

func NewBoundFunctionValue(
	interpreter *Interpreter,
	function FunctionValue,
	self *MemberAccessibleValue,
) BoundFunctionValue {

	common.UseMemory(interpreter, common.BoundFunctionValueMemoryUsage)

	return BoundFunctionValue{
		Function: function,
		Self:     self,
	}
}

func (BoundFunctionValue) IsValue() {}

func (f BoundFunctionValue) String() string {
	return f.RecursiveString(SeenReferences{})
}

func (f BoundFunctionValue) RecursiveString(seenReferences SeenReferences) string {
	return f.Function.RecursiveString(seenReferences)
}

func (f BoundFunctionValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	return f.Function.MeteredString(memoryGauge, seenReferences)
}

func (f BoundFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitBoundFunctionValue(interpreter, f)
}

func (f BoundFunctionValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (f BoundFunctionValue) StaticType(inter *Interpreter) StaticType {
	return f.Function.StaticType(inter)
}

func (BoundFunctionValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (BoundFunctionValue) isFunctionValue() {}

func (f BoundFunctionValue) FunctionType() *sema.FunctionType {
	return f.Function.FunctionType()
}

func (f BoundFunctionValue) invoke(invocation Invocation) Value {
	invocation.Self = f.Self
	return f.Function.invoke(invocation)
}

func (f BoundFunctionValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	return f.Function.ConformsToStaticType(
		interpreter,
		locationRange,
		results,
	)
}

func (f BoundFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (BoundFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (BoundFunctionValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (f BoundFunctionValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return f
}

func (f BoundFunctionValue) Clone(_ *Interpreter) Value {
	return f
}

func (BoundFunctionValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}
