import XCTest
import GraphQLDocuments
import GraphQLSyntax

struct MacroAvatarFragmentData {
    let avatarUrl: String
}

@GraphQLFragment("""
    fragment Avatar_user on User { avatarUrl }
    """)
enum MacroAvatarFragment {}

final class GraphQLMacroTests: XCTestCase {
    func testMacroConnectsDeclarationToGeneratedData() {
        let data: MacroAvatarFragment.Data = MacroAvatarFragmentData(avatarUrl: "/avatar.svg")
        XCTAssertEqual(data.avatarUrl, "/avatar.svg")
        XCTAssertEqual(MacroAvatarFragment.document, "fragment Avatar_user on User { avatarUrl }")
    }

    func testExtractionUsesSwiftSyntaxAndPreservesLocation() throws {
        let source = #"""
        // @GraphQLFragment("fake") enum Fake {}
        @GraphQLFragment("""
            fragment Avatar_user on User { avatarUrl }
            """)
        enum AvatarFragment {}
        """#
        let documents = try extractDocuments(source: source, file: "Avatar.swift")
        XCTAssertEqual(documents.count, 1)
        XCTAssertEqual(documents[0].line, 3)
        XCTAssertEqual(documents[0].column, 5)
        XCTAssertEqual(documents[0].native.declaration, "AvatarFragment")
    }

    func testRejectsInterpolation() {
        let source = #"""
        @GraphQLFragment("""
            fragment Avatar_user on User { \(field) }
            """)
        enum AvatarFragment {}
        """#
        XCTAssertThrowsError(try extractDocuments(source: source, file: "Avatar.swift")) { error in
            XCTAssertTrue(String(describing: error).contains("without interpolation"))
        }
    }
}
