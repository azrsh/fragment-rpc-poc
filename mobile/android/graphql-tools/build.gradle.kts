plugins {
    id("org.jetbrains.kotlin.jvm")
    application
}
kotlin { jvmToolchain(17) }
application { mainClass.set("dev.fragmentrpc.codegen.ExtractKt") }
dependencies {
    implementation("org.jetbrains.kotlin:kotlin-compiler-embeddable:2.2.21")
    implementation("com.google.devtools.ksp:symbol-processing-api:2.2.21-2.0.4")
    implementation("com.squareup:kotlinpoet:2.2.0")
    implementation("com.google.code.gson:gson:2.13.1")
}
