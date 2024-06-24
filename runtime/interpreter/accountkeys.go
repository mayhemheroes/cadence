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
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// AuthAccountKeys

var authAccountKeysTypeID = sema.AuthAccountKeysType.ID()
var authAccountKeysStaticType StaticType = PrimitiveStaticTypeAuthAccountKeys

// NewAuthAccountKeysValue constructs a AuthAccount.Keys value.
func NewAuthAccountKeysValue(
	gauge common.MemoryGauge,
	address AddressValue,
	addFunction FunctionValue,
	getFunction FunctionValue,
	revokeFunction FunctionValue,
	forEachFunction FunctionValue,
	getKeysCount AccountKeysCountGetter,
) Value {

	fields := map[string]Value{
		sema.AccountKeysTypeAddFunctionName:     addFunction,
		sema.AccountKeysTypeGetFunctionName:     getFunction,
		sema.AccountKeysTypeRevokeFunctionName:  revokeFunction,
		sema.AccountKeysTypeForEachFunctionName: forEachFunction,
	}

	computeField := func(name string, _ *Interpreter, _ LocationRange) Value {
		switch name {
		case sema.AccountKeysTypeCountFieldName:
			return getKeysCount()
		}
		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AuthAccountKeysStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("AuthAccount.Keys(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		authAccountKeysTypeID,
		authAccountKeysStaticType,
		nil,
		fields,
		computeField,
		nil,
		stringer,
	)
}

// PublicAccountKeys

var publicAccountKeysTypeID = sema.PublicAccountKeysType.ID()
var publicAccountKeysStaticType StaticType = PrimitiveStaticTypePublicAccountKeys

// NewPublicAccountKeysValue constructs a PublicAccount.Keys value.
func NewPublicAccountKeysValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getFunction FunctionValue,
	forEachFunction FunctionValue,
	getKeysCount AccountKeysCountGetter,
) Value {

	fields := map[string]Value{
		sema.AccountKeysTypeGetFunctionName:     getFunction,
		sema.AccountKeysTypeForEachFunctionName: forEachFunction,
	}

	computeField := func(name string, _ *Interpreter, _ LocationRange) Value {
		switch name {
		case sema.AccountKeysTypeCountFieldName:
			return getKeysCount()
		}
		return nil
	}
	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.PublicAccountKeysStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("PublicAccount.Keys(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		publicAccountKeysTypeID,
		publicAccountKeysStaticType,
		nil,
		fields,
		computeField,
		nil,
		stringer,
	)
}

type AccountKeysCountGetter func() UInt64Value
