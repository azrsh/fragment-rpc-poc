import Foundation
import SwiftParser
import SwiftSyntax

public struct DocumentError: Error, CustomStringConvertible {
    public let description: String
    public init(_ description: String) { self.description = description }
}

public func documentLiteral(_ attribute: AttributeSyntax) throws -> StringLiteralExprSyntax {
    guard case .argumentList(let arguments) = attribute.arguments,
          arguments.count == 1,
          let literal = arguments.first?.expression.as(StringLiteralExprSyntax.self),
          literal.segments.allSatisfy({ $0.is(StringSegmentSyntax.self) }) else {
        throw DocumentError("GraphQL declarations require one literal string without interpolation")
    }
    return literal
}

public func documentText(_ literal: StringLiteralExprSyntax) throws -> String {
    // Restrict the authoring form so the extractor and Swift evaluate identical documents.
    guard literal.openingQuote.text == "\"\"\"", literal.openingPounds == nil else {
        throw DocumentError("Use a multiline triple-quoted GraphQL literal")
    }
    let value = literal.segments.compactMap { $0.as(StringSegmentSyntax.self)?.content.text }.joined()
    guard !value.contains("\\") else {
        throw DocumentError("Escapes are not supported in embedded GraphQL; use a .graphql file for escaped values")
    }
    return value
}

public struct NativeDeclaration: Encodable {
    public let language = "swift"
    public let declaration: String
    public let kind: String
}

public struct ExtractedDocument: Encodable {
    public let name: String
    public let body: String
    public let line: Int
    public let column: Int
    public let native: NativeDeclaration
}

final class DocumentVisitor: SyntaxVisitor {
    let converter: SourceLocationConverter
    let file: String
    var documents: [ExtractedDocument] = []
    var failures: [String] = []

    init(file: String, tree: SourceFileSyntax) {
        self.file = file
        converter = SourceLocationConverter(fileName: file, tree: tree)
        super.init(viewMode: .sourceAccurate)
    }

    override func visit(_ node: EnumDeclSyntax) -> SyntaxVisitorContinueKind {
        for attribute in node.attributes.compactMap({ $0.as(AttributeSyntax.self) }) {
            let marker = attribute.attributeName.trimmedDescription
            guard marker == "GraphQLFragment" || marker == "GraphQLQuery" else { continue }
            do {
                let literal = try documentLiteral(attribute)
                let body = try documentText(literal)
                let start = converter.location(for: literal.segments.positionAfterSkippingLeadingTrivia)
                documents.append(ExtractedDocument(name: file, body: body, line: start.line, column: start.column,
                    native: NativeDeclaration(declaration: node.name.text, kind: marker == "GraphQLFragment" ? "fragment" : "query")))
            } catch {
                let start = converter.location(for: attribute.positionAfterSkippingLeadingTrivia)
                failures.append("\(file):\(start.line):\(start.column): error: \(error)")
            }
        }
        return .visitChildren
    }
}

public func extractDocuments(source: String, file: String) throws -> [ExtractedDocument] {
    let tree = Parser.parse(source: source)
    let visitor = DocumentVisitor(file: file, tree: tree)
    visitor.walk(tree)
    if !visitor.failures.isEmpty { throw DocumentError(visitor.failures.joined(separator: "\n")) }
    return visitor.documents
}
