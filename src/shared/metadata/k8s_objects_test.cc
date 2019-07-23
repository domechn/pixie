#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include "src/shared/metadata/k8s_objects.h"

namespace pl {
namespace md {

TEST(PodInfo, basic_accessors) {
  PodInfo pod_info("123", "pl", "pod1", PodQOSClass::kGuaranteed);
  pod_info.set_start_time_ns(123);
  pod_info.set_stop_time_ns(256);

  EXPECT_EQ("123", pod_info.uid());
  EXPECT_EQ("pl", pod_info.ns());
  EXPECT_EQ("pod1", pod_info.name());
  EXPECT_EQ(PodQOSClass::kGuaranteed, pod_info.qos_class());

  EXPECT_EQ(123, pod_info.start_time_ns());
  EXPECT_EQ(256, pod_info.stop_time_ns());

  EXPECT_EQ(K8sObjectType::kPod, pod_info.type());
}

TEST(PodInfo, debug_string) {
  PodInfo pod_info("123", "pl", "pod1", PodQOSClass::kGuaranteed);
  for (int i = 0; i < 5; ++i) {
    EXPECT_EQ(absl::Substitute("$0<Pod:ns=pl:name=pod1:uid=123:state=R>", Indent(i)),
              pod_info.DebugString(i));
  }

  pod_info.set_stop_time_ns(1000);
  EXPECT_EQ("<Pod:ns=pl:name=pod1:uid=123:state=S>", pod_info.DebugString());
}

TEST(PodInfo, add_delete_containers) {
  PodInfo pod_info("123", "pl", "pod1", PodQOSClass::kGuaranteed);
  pod_info.AddContainer("ABCD");
  pod_info.AddContainer("ABCD2");
  pod_info.AddContainer("ABCD3");
  pod_info.RmContainer("ABCD");

  EXPECT_THAT(pod_info.containers(), testing::UnorderedElementsAre("ABCD2", "ABCD3"));

  pod_info.RmContainer("ABCD3");
  EXPECT_THAT(pod_info.containers(), testing::UnorderedElementsAre("ABCD2"));
}

TEST(PodInfo, clone) {
  PodInfo pod_info("123", "pl", "pod1", PodQOSClass::kBurstable);
  pod_info.set_start_time_ns(123);
  pod_info.set_stop_time_ns(256);
  pod_info.AddContainer("ABCD");
  pod_info.AddContainer("ABCD2");

  EXPECT_EQ(PodQOSClass::kBurstable, pod_info.qos_class());

  std::unique_ptr<PodInfo> cloned(static_cast<PodInfo*>(pod_info.Clone().release()));
  EXPECT_EQ(cloned->uid(), pod_info.uid());
  EXPECT_EQ(cloned->name(), pod_info.name());
  EXPECT_EQ(cloned->ns(), pod_info.ns());
  EXPECT_EQ(cloned->qos_class(), pod_info.qos_class());

  EXPECT_EQ(cloned->start_time_ns(), pod_info.start_time_ns());
  EXPECT_EQ(cloned->stop_time_ns(), pod_info.stop_time_ns());

  EXPECT_EQ(cloned->type(), pod_info.type());
  EXPECT_EQ(cloned->containers(), pod_info.containers());
}

TEST(ContainerInfo, pod_id) {
  ContainerInfo cinfo("container1", 128 /*start_time*/);

  EXPECT_EQ("", cinfo.pod_id());
  cinfo.set_pod_id("pod1");
  EXPECT_EQ("pod1", cinfo.pod_id());
}

TEST(ContainerInfo, debug_string) {
  ContainerInfo cinfo("container1", 128);
  for (int i = 0; i < 5; ++i) {
    EXPECT_EQ(absl::Substitute("$0<Container:cid=container1:pod_id=:state=R>", Indent(i)),
              cinfo.DebugString(i));
  }

  cinfo.set_stop_time_ns(1000);
  EXPECT_EQ("<Container:cid=container1:pod_id=:state=S>", cinfo.DebugString());
}

TEST(ContainerInfo, add_delete_pids) {
  ContainerInfo cinfo("container1", 128 /*start_time*/);
  cinfo.set_pod_id("pod1");

  cinfo.AddUPID(UPID(1, 1, 123));
  cinfo.AddUPID(UPID(1, 2, 123));
  cinfo.AddUPID(UPID(1, 2, 123));
  cinfo.AddUPID(UPID(1, 5, 123));

  EXPECT_THAT(cinfo.active_upids(),
              testing::UnorderedElementsAre(UPID(1, 1, 123), UPID(1, 2, 123), UPID(1, 5, 123)));
  EXPECT_THAT(cinfo.inactive_upids(), testing::UnorderedElementsAre());

  cinfo.DeactivateUPID(UPID(1, 2, 123));
  EXPECT_THAT(cinfo.active_upids(),
              testing::UnorderedElementsAre(UPID(1, 1, 123), UPID(1, 5, 123)));
  EXPECT_THAT(cinfo.inactive_upids(), testing::UnorderedElementsAre(UPID(1, 2, 123)));
}

TEST(ContainerInfo, deactive_non_existing_pid_ignored) {
  ContainerInfo cinfo("container1", 128 /*start_time*/);
  cinfo.set_pod_id("pod1");
  cinfo.DeactivateUPID(UPID(1, 3, 123));

  EXPECT_THAT(cinfo.active_upids(), testing::UnorderedElementsAre());
  EXPECT_THAT(cinfo.inactive_upids(), testing::UnorderedElementsAre());
}

TEST(ContainerInfo, clone) {
  ContainerInfo orig("container1", 128 /*start_time*/);
  orig.set_pod_id("pod1");

  orig.AddUPID(UPID(1, 0, 123));
  orig.AddUPID(UPID(1, 1, 123));
  orig.AddUPID(UPID(1, 15, 123));

  auto cloned = orig.Clone();

  EXPECT_EQ(cloned->pod_id(), orig.pod_id());
  EXPECT_EQ(cloned->cid(), orig.cid());
  EXPECT_EQ(cloned->active_upids(), orig.active_upids());
  EXPECT_EQ(cloned->start_time_ns(), orig.start_time_ns());
  EXPECT_EQ(cloned->stop_time_ns(), orig.stop_time_ns());
}

}  // namespace md
}  // namespace pl
