use std::env;
use std::path::PathBuf;
use std::process::Command;

fn main() {
    let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap());
    let open62541_dir = PathBuf::from("third_party/open62541");
    let open62541_build_dir = out_dir.join("open62541_build");
    let open62541_install_dir = out_dir.join("open62541_install");

    // Create build and install directories
    std::fs::create_dir_all(&open62541_build_dir).unwrap();
    std::fs::create_dir_all(&open62541_install_dir).unwrap();

    // Configure open62541 with CMake
    let cmake_status = Command::new("cmake")
        .current_dir(&open62541_build_dir)
        .arg(open62541_dir.to_str().unwrap())
        .arg(format!("-DCMAKE_INSTALL_PREFIX={}", open62541_install_dir.to_str().unwrap()))
        .arg("-DUA_ENABLE_AMALGAMATION=ON") // Build as a single file
        .arg("-DUA_BUILD_EXAMPLES=OFF")
        .arg("-DUA_BUILD_UNIT_TESTS=OFF")
        .arg("-DUA_BUILD_FUZZING_TESTS=OFF")
        .arg("-DUA_BUILD_BENCHMARKS=OFF")
        .arg("-DUA_ENABLE_DISCOVERY=ON")
        .arg("-DUA_ENABLE_SUBSCRIPTIONS=ON")
        .arg("-DUA_ENABLE_METHODCALLS=ON")
        .arg("-DUA_ENABLE_NODEMANAGEMENT=ON")
        .arg("-DUA_ENABLE_ENCRYPTION=ON")
        .arg("-DUA_ENABLE_HISTORIZING=ON")
        .arg("-DUA_ENABLE_PUBSUB=ON")
        .arg("-DUA_ENABLE_MICRO_EMB_UA_TYPES=ON")
        .output()
        .expect("Failed to run cmake");

    if !cmake_status.status.success() {
        panic!("cmake failed: {:?}", cmake_status);
    }

    // Build open62541
    let make_status = Command::new("cmake")
        .current_dir(&open62541_build_dir)
        .arg("--build")
        .arg(".")
        .arg("--target")
        .arg("install")
        .output()
        .expect("Failed to run make");

    if !make_status.status.success() {
        panic!("make failed: {:?}", make_status);
    }

    // Link against the compiled library
    println!("cargo:rustc-link-search=native={}/lib", open62541_install_dir.to_str().unwrap());
    println!("cargo:rustc-link-lib=static=open62541");

    // Generate bindings
    let bindings = bindgen::Builder::default()
        .header(open62541_install_dir.join("include/open62541/open62541.h").to_str().unwrap())
        .parse_callbacks(Box::new(<dyn bindgen::callbacks::ParseCallbacks>::default()))
        .generate()
        .expect("Unable to generate bindings");

    bindings
        .write_to_file(out_dir.join("bindings.rs"))
        .expect("Couldn't write bindings!");
}
