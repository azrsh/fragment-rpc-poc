import SwiftCompilerPlugin
import SwiftSyntax
import SwiftSyntaxMacros
import GraphQLSyntax

public struct GraphQLMacro: MemberMacro {
    public static func expansion(of node: AttributeSyntax, providingMembersOf declaration: some DeclGroupSyntax,
                                 in context: some MacroExpansionContext) throws -> [DeclSyntax] {
        guard let declaration = declaration.as(EnumDeclSyntax.self), declaration.memberBlock.members.isEmpty else {
            throw DocumentError("Attach GraphQL declarations to an empty enum")
        }
        let literal = try documentLiteral(node)
        _ = try documentText(literal)
        let generated = declaration.name.text + "Data"
        return [
            "typealias Data = \(raw: generated)",
            "static let document: String = \(literal)",
        ]
    }
}

@main
struct GraphQLPlugin: CompilerPlugin {
    let providingMacros: [Macro.Type] = [GraphQLMacro.self]
}
