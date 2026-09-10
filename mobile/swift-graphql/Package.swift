// swift-tools-version: 6.0
import PackageDescription
import CompilerPluginSupport

let package = Package(
    name: "GraphQLDocuments",
    platforms: [.macOS(.v13), .iOS(.v17)],
    products: [
        .library(name: "GraphQLDocuments", targets: ["GraphQLDocuments"]),
        .executable(name: "extract-graphql", targets: ["ExtractGraphQL"]),
    ],
    dependencies: [
        .package(url: "https://github.com/swiftlang/swift-syntax.git", exact: "602.0.0"),
    ],
    targets: [
        .target(name: "GraphQLSyntax", dependencies: [
            .product(name: "SwiftSyntax", package: "swift-syntax"),
            .product(name: "SwiftParser", package: "swift-syntax"),
        ]),
        .macro(name: "GraphQLMacros", dependencies: [
            "GraphQLSyntax",
            .product(name: "SwiftCompilerPlugin", package: "swift-syntax"),
            .product(name: "SwiftSyntaxMacros", package: "swift-syntax"),
        ]),
        .target(name: "GraphQLDocuments", dependencies: ["GraphQLMacros"]),
        .executableTarget(name: "ExtractGraphQL", dependencies: ["GraphQLSyntax"]),
        .testTarget(name: "GraphQLMacrosTests", dependencies: [
            "GraphQLDocuments", "GraphQLSyntax",
        ]),
    ]
)
