// ABOUTME: CLI validation commands

use crate::error::Result;
use crate::validation::{PerformanceValidator, PlatformValidator};

pub fn run_performance_validation() -> Result<()> {
    println!("🔍 Running vector search performance validation...");

    let validator = PerformanceValidator::new();
    let results = validator.validate_vector_search_performance()?;

    println!("\n📊 Performance Validation Results:");
    println!("=================================");

    for result in &results {
        let status = if result.success { "✅ PASS" } else { "❌ FAIL" };
        println!("{} {} ({}ms)", status, result.test_name, result.duration_ms);
        println!("   {}", result.details);
        println!();
    }

    // Check overall success
    let all_passed = results.iter().all(|r| r.success);
    if all_passed {
        println!("🎉 All performance tests passed!");
    } else {
        println!("⚠️  Some performance tests failed.");
    }

    Ok(())
}

pub fn run_platform_validation() -> Result<()> {
    println!("🔍 Running cross-platform compatibility validation...");

    let validator = PlatformValidator::new();
    let results = validator.validate_platform_compatibility()?;

    println!("\n📊 Platform Compatibility Results:");
    println!("===================================");

    for result in &results {
        let status = if result.success { "✅ PASS" } else { "❌ FAIL" };
        println!("{} {} ({})", status, result.test_name, result.platform);
        println!("   {}", result.details);
        println!();
    }

    // Check overall success
    let all_passed = results.iter().all(|r| r.success);
    if all_passed {
        println!("🎉 All platform tests passed!");
    } else {
        println!("⚠️  Some platform tests failed.");
    }

    Ok(())
}