load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")
load("@com_github_grpc_grpc//:bazel/grpc_deps.bzl", "grpc_deps")
load("@io_bazel_rules_docker//go:image.bzl", _go_image_repos = "repositories")
load("@io_bazel_rules_docker//cc:image.bzl", _cc_image_repos = "repositories")
load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
)
load("@distroless//package_manager:package_manager.bzl", "dpkg_list", "dpkg_src", "package_manager_repositories")

def _go_setup():
    go_rules_dependencies()
    go_register_toolchains()
    gazelle_dependencies()

# Sets up package manager which we use build deploy images.
def _package_manager_setup():
    package_manager_repositories()

    dpkg_src(
        name = "debian_stretch",
        arch = "amd64",
        distro = "stretch",
        sha256 = "9aea0e4c9ce210991c6edcb5370cb9b11e9e554a0f563e7754a4028a8fd0cb73",
        snapshot = "20171101T160520Z",
        url = "http://snapshot.debian.org/archive",
    )

    dpkg_list(
        name = "package_bundle",
        packages = [
            "libc6",
            "libelf1",
            "liblzma5",
            "libtinfo5",
            "libunwind8",
            "zlib1g",
        ],
        sources = ["@debian_stretch//file:Packages.json"],
    )

def _docker_setup():
    _go_image_repos()
    _cc_image_repos()
    _package_manager_setup()

    # Import NGINX repo.
    container_pull(
        name = "nginx_base",
        digest = "sha256:9ad0746d8f2ea6df3a17ba89eca40b48c47066dfab55a75e08e2b70fc80d929e",
        registry = "index.docker.io",
        repository = "library/nginx",
    )

    # Import CC base image
    container_pull(
        name = "cc_base",
        # From : March 27, 2019
        digest = "sha256:482e7efb3245ded60e9ced05909551fc14d39b47e2cc643830f4466010c25372",
        registry = "gcr.io",
        repository = "distroless/cc",
    )

def pl_workspace_setup():
    _go_setup()
    buildifier_dependencies()
    grpc_deps()
    _docker_setup()
