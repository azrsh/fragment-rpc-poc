import Foundation
import GraphQLSyntax

do {
    var documents: [ExtractedDocument] = []
    for path in CommandLine.arguments.dropFirst() {
        documents += try extractDocuments(source: String(contentsOfFile: path, encoding: .utf8), file: path)
    }
    FileHandle.standardOutput.write(try JSONEncoder().encode(documents))
} catch {
    FileHandle.standardError.write(Data("\(error)\n".utf8))
    exit(1)
}
