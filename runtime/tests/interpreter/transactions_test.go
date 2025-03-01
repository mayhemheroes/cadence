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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretTransactions(t *testing.T) {

	t.Parallel()

	t.Run("NoPrepareFunction", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {
              let x = 1 + 2
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("SetTransactionField", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              let y = self.x + 1
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("PreConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            pre {
              self.x > 1
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("FailingPreConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            pre {
              self.x > 10
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPre,
			conditionErr.ConditionKind,
		)
	})

	t.Run("PostConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              self.x = 10
            }

            post {
              self.x == 10
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)
	})

	t.Run("FailingPostConditions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {

            var x: Int

            prepare() {
              self.x = 5
            }

            execute {
              self.x = 10
            }

            post {
              self.x == 5
            }
          }
        `)

		err := inter.InvokeTransaction(0)
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t,
			ast.ConditionKindPost,
			conditionErr.ConditionKind,
		)
	})

	t.Run("MultipleTransactions", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {
              let x = 1 + 2
            }
          }

          transaction {
            execute {
              let y = 3 + 4
            }
          }
        `)

		// first transaction
		err := inter.InvokeTransaction(0)
		assert.NoError(t, err)

		// second transaction
		err = inter.InvokeTransaction(1)
		assert.NoError(t, err)

		// third transaction is not declared
		err = inter.InvokeTransaction(2)
		assert.IsType(t, interpreter.TransactionNotDeclaredError{}, err)
	})

	t.Run("TooFewArguments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            prepare(signer: AuthAccount) {}
          }
        `)

		err := inter.InvokeTransaction(0)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("TooManyArguments", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          transaction {
            execute {}
          }

          transaction {
            prepare(signer: AuthAccount) {}

            execute {}
          }
        `)

		signer1 := newTestAuthAccountValue(
			nil,
			interpreter.AddressValue{0, 0, 0, 0, 0, 0, 0, 1},
		)
		signer2 := newTestAuthAccountValue(
			nil,
			interpreter.AddressValue{0, 0, 0, 0, 0, 0, 0, 2},
		)

		// first transaction
		err := inter.InvokeTransaction(0, signer1)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)

		// second transaction
		err = inter.InvokeTransaction(0, signer1, signer2)
		assert.IsType(t, interpreter.ArgumentCountError{}, err)
	})

	t.Run("Parameters", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let values: [AnyStruct] = []

          transaction(x: Int, y: Bool) {

            prepare(signer: AuthAccount) {
              values.append(signer.address)
              values.append(y)
              values.append(x)
            }
          }
        `)

		arguments := []interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.BoolValue(true),
		}

		prepareArguments := []interpreter.Value{
			newTestAuthAccountValue(
				nil,
				interpreter.AddressValue{},
			),
		}

		arguments = append(arguments, prepareArguments...)

		err := inter.InvokeTransaction(0, arguments...)
		assert.NoError(t, err)

		values := inter.Globals.Get("values").GetValue()

		require.IsType(t, &interpreter.ArrayValue{}, values)

		AssertValueSlicesEqual(
			t,
			inter,
			[]interpreter.Value{
				interpreter.AddressValue{},
				interpreter.BoolValue(true),
				interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			arrayElements(inter, values.(*interpreter.ArrayValue)),
		)
	})
}
