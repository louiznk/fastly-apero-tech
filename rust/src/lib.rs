pub mod application;

#[cfg(feature = "fastly")]
pub mod fastlyadapter;

#[cfg(feature = "native-http")]
pub mod httpadapter;
