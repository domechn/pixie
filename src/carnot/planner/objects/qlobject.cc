/*
 * Copyright 2018- The Pixie Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#include <vector>

#include "src/carnot/planner/objects/collection_object.h"
#include "src/carnot/planner/objects/dataframe.h"
#include "src/carnot/planner/objects/expr_object.h"
#include "src/carnot/planner/objects/funcobject.h"
#include "src/carnot/planner/objects/qlobject.h"

namespace px {
namespace carnot {
namespace planner {
namespace compiler {
std::string QLObjectTypeString(QLObjectType type) {
  return std::string(absl::StripPrefix(absl::AsciiStrToLower(magic_enum::enum_name(type)), "k"));
}

void QLObject::AddSubscriptMethod(std::shared_ptr<FuncObject> func_object) {
  DCHECK_EQ(func_object->name(), kSubscriptMethodName);
  DCHECK(func_object->arguments() == std::vector<std::string>{"key"})
      << absl::StrJoin(func_object->arguments(), ",");
  AddMethod(kSubscriptMethodName, func_object);
}

StatusOr<std::shared_ptr<QLObject>> QLObject::GetAttribute(const pypa::AstPtr& ast,
                                                           std::string_view attr) const {
  if (HasMethod(attr)) {
    return GetMethod(attr);
  }
  if (!HasNonMethodAttribute(attr)) {
    return CreateAstError(ast, "'$0' object has no attribute '$1'", name(), attr);
  }
  return GetAttributeImpl(ast, attr);
}

Status QLObject::AssignAttribute(std::string_view attr_name, QLObjectPtr object) {
  if (!CanAssignAttribute(attr_name)) {
    return CreateError("Cannot assign attribute $0 to object of type $1", attr_name, name());
  }
  attributes_[attr_name] = object;
  return Status::OK();
}

StatusOr<QLObjectPtr> QLObject::FromIRNode(IRNode* node, ASTVisitor* ast_visitor) {
  if (Match(node, Operator())) {
    return Dataframe::Create(static_cast<OperatorIR*>(node), ast_visitor);
  } else if (Match(node, Expression())) {
    return ExprObject::Create(static_cast<ExpressionIR*>(node), ast_visitor);
  } else {
    return node->CreateIRNodeError("Could not create QL object from IRNode of type $0",
                                   node->type_string());
  }
}

Status QLObject::SetDocString(const std::string& doc_string) {
  // TODO(PP-2142) Support non ExprObject nodes to pass strings around so we can
  // store it in __doc__.
  doc_string_ = doc_string;
  return Status::OK();
}

Status QLObject::SetDocString(QLObjectPtr doc_string) {
  // TODO(PP-2142) Redo the doc_string requirements so StringIR isn't a requirement.
  PL_RETURN_IF_ERROR(AssignAttribute(kDocStringAttributeName, doc_string));
  DCHECK(doc_string->HasNode());
  DCHECK_EQ(doc_string->node()->type(), IRNodeType::kString);
  doc_string_ = static_cast<StringIR*>(doc_string->node())->str();
  return Status::OK();
}

}  // namespace compiler
}  // namespace planner
}  // namespace carnot
}  // namespace px
