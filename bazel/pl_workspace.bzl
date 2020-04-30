load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")
load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps")
load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)
load("@io_bazel_rules_docker//go:image.bzl", _go_image_repos = "repositories")
load("@io_bazel_rules_docker//java:image.bzl", _java_image_repos = "repositories")
load("@io_bazel_rules_docker//container:container.bzl", "container_pull")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")
load("@distroless//package_manager:package_manager.bzl", "package_manager_repositories")
load("@distroless//package_manager:dpkg.bzl", "dpkg_list", "dpkg_src")
load("@io_bazel_rules_k8s//k8s:k8s.bzl", "k8s_repositories")
load("@io_bazel_rules_k8s//k8s:k8s_go_deps.bzl", k8s_go_deps = "deps")

# Sets up package manager which we use build deploy images.
def _package_manager_setup():
    package_manager_repositories()

    dpkg_src(
        name = "debian_buster",
        arch = "amd64",
        distro = "buster",
        sha256 = "bd1bed6b19bf173d60ac130edee47087203e873f3b0981f5987f77a91a2cba85",
        snapshot = "20190716T085419Z",
        url = "http://snapshot.debian.org/archive",
    )

    dpkg_list(
        name = "package_bundle",
        packages = [
            "libc6",
            "libelf1",
            "liblzma5",
            "libtinfo5",
            "zlib1g",
            "libsasl2-2",
            "libssl1.1",
            "libgcc1",
        ],
        sources = ["@debian_buster//file:Packages.json"],
    )

def _docker_images_setup():
    _go_image_repos()
    _java_image_repos()

    # Import NGINX repo.
    container_pull(
        name = "nginx_base",
        digest = "sha256:204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad",
        registry = "index.docker.io",
        repository = "library/nginx",
    )

    container_pull(
        name = "base_image",
        digest = "sha256:e37cf3289c1332c5123cbf419a1657c8dad0811f2f8572433b668e13747718f8",
        registry = "gcr.io",
        repository = "distroless/base",
    )

    container_pull(
        name = "base_image_debug",
        digest = "sha256:f989df6099c5efb498021c7f01b74f484b46d2f5e1cdb862e508569d87569f2b",
        registry = "gcr.io",
        repository = "distroless/base",
    )

def _artifacts_setup():
    http_file(
        name = "linux_headers_4_14_176_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-4.14.176-trimmed-pl3.tar.gz"],
        sha256 = "67a59f55cb8592ed03719fedb925cdf7a2dc8529fcf9ab1002405540a855212c",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_4_15_18_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-4.15.18-trimmed-pl3.tar.gz"],
        sha256 = "0a82dea437d1798a88df95498892f9d14a5158f25184f42a90c5ce093645529d",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_4_16_18_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-4.16.18-trimmed-pl3.tar.gz"],
        sha256 = "738362e58aa11a51ff292c0520dd36ddfecc9ca1494c8b2841d01e51ceaf769a",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_4_17_19_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-4.17.19-trimmed-pl3.tar.gz"],
        sha256 = "38855fd5786fd459d92ce7193fc7379af2c1a7480e0bac95b0ba291fc08b4eea",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_4_18_20_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-4.18.20-trimmed-pl3.tar.gz"],
        sha256 = "efff57e9642ad968ceee4b7c0f7387fd2507499c12bda79b850b40fa35951265",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_4_19_118_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-4.19.118-trimmed-pl3.tar.gz"],
        sha256 = "43253ad88cc276b293c0cbe35b684e5462af2ffa180775c0973b0e278b4f9ee6",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_4_20_17_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-4.20.17-trimmed-pl3.tar.gz"],
        sha256 = "baa9631a9916330879a8c487b5a9ac0f73ba1b53e38283c6f65350a9f2f63ba6",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_5_0_21_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-5.0.21-trimmed-pl3.tar.gz"],
        sha256 = "848a1135a69763bac3afff1c1bf9ac3ba63d04026479d146936d701619b44bb1",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_5_1_21_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-5.1.21-trimmed-pl3.tar.gz"],
        sha256 = "4750ca03b38301f3627b47a4dc5690e6d5ba641c18a6eafdb37cb8f86614572f",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_5_2_21_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-5.2.21-trimmed-pl3.tar.gz"],
        sha256 = "36c90df582a85c865e7fefe99db51fd82117c32bdd72452da5a47e73da8b7355",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_5_3_18_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-5.3.18-trimmed-pl3.tar.gz"],
        sha256 = "2e4b3eff995177122c4f28096f5a9a815fb2a1d0c025dc5340d6d86a9a7796e9",
        downloaded_file_path = "linux_headers.tar.gz",
    )

    http_file(
        name = "linux_headers_5_4_35_tar_gz",
        urls = ["https://storage.googleapis.com/pl-infra-dev-artifacts/linux-headers-5.4.35-trimmed-pl3.tar.gz"],
        sha256 = "f371fc16c3542b6a7a47788693f00e743ec82996925c3dee7123c588e59210f7",
        downloaded_file_path = "linux_headers.tar.gz",
    )

# TODO(zasgar): remove this when downstream bugs relying on bazel version are removed.
def _impl(repository_ctx):
    bazel_verision_for_upb = "bazel_version = \"" + native.bazel_version + "\""
    bazel_version_for_foreign_cc = "BAZEL_VERSION = \"" + native.bazel_version + "\""
    repository_ctx.file("bazel_version.bzl", bazel_verision_for_upb)
    repository_ctx.file("def.bzl", bazel_version_for_foreign_cc)
    repository_ctx.file("BUILD", "")

bazel_version_repository = repository_rule(
    implementation = _impl,
    local = True,
)

def pl_workspace_setup():
    gazelle_dependencies()
    buildifier_dependencies()
    grpc_deps()

    bazel_version_repository(
        name = "bazel_version",
    )

    container_repositories()

    k8s_repositories()
    k8s_go_deps()

    _package_manager_setup()
    _docker_images_setup()
    _artifacts_setup()
