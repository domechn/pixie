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

#include <sole.hpp>

#include "src/vizier/services/agent/pem/pem_manager.h"

#include "src/common/base/base.h"
#include "src/common/signal/signal.h"
#include "src/shared/version/version.h"

DEFINE_string(nats_url, gflags::StringFromEnv("PL_NATS_URL", "pl-nats"),
              "The host address of the nats cluster");
DEFINE_string(pod_name, gflags::StringFromEnv("PL_POD_NAME", ""),
              "The name of the POD the PEM is running on");
DEFINE_string(host_ip, gflags::StringFromEnv("PL_HOST_IP", ""),
              "The IP of the host this service is running on");

using ::px::vizier::agent::Manager;
using ::px::vizier::agent::PEMManager;

class AgentDeathHandler : public px::FatalErrorHandlerInterface {
 public:
  AgentDeathHandler() = default;
  void OnFatalError() const override {
    // Stack trace will print automatically; any additional state dumps can be done here.
    // Note that actions here must be async-signal-safe and must not allocate memory.
  }
};

// Signal handlers for graceful termination.
class AgentTerminationHandler {
 public:
  // This list covers signals that are handled gracefully.
  static constexpr auto kSignals = ::px::MakeArray(SIGINT, SIGQUIT, SIGTERM, SIGHUP);

  static void InstallSignalHandlers() {
    for (size_t i = 0; i < kSignals.size(); ++i) {
      signal(kSignals[i], AgentTerminationHandler::OnTerminate);
    }
  }

  static void set_manager(Manager* manager) { manager_ = manager; }

  static void OnTerminate(int signum) {
    if (manager_ != nullptr) {
      LOG(INFO) << "Trying to gracefully stop agent manager";
      auto s = manager_->Stop(std::chrono::seconds{5});
      if (!s.ok()) {
        LOG(ERROR) << "Failed to gracefully stop agent manager, it will terminate shortly.";
      }
      exit(signum);
    }
  }

 private:
  inline static Manager* manager_ = nullptr;
};

int main(int argc, char** argv) {
  px::EnvironmentGuard env_guard(&argc, argv);

  AgentDeathHandler err_handler;

  // This covers signals such as SIGSEGV and other fatal errors.
  // We print the stack trace and die.
  auto signal_action = std::make_unique<px::SignalAction>();
  signal_action->RegisterFatalErrorHandler(err_handler);

  // Install signal handlers where graceful exit is possible.
  AgentTerminationHandler::InstallSignalHandlers();

  sole::uuid agent_id = sole::uuid4();
  LOG(INFO) << absl::Substitute("Pixie PEM. Version: $0, id: $1", px::VersionInfo::VersionString(),
                                agent_id.str());

  if (FLAGS_host_ip.length() == 0) {
    LOG(FATAL) << "The HOST_IP must be specified";
  }
  auto manager = PEMManager::Create(agent_id, FLAGS_pod_name, FLAGS_host_ip, FLAGS_nats_url)
                     .ConsumeValueOrDie();

  AgentTerminationHandler::set_manager(manager.get());

  PL_CHECK_OK(manager->Run());
  PL_CHECK_OK(manager->Stop(std::chrono::seconds{5}));

  // Clear the manager, because it has been stopped.
  AgentTerminationHandler::set_manager(nullptr);

  return 0;
}
