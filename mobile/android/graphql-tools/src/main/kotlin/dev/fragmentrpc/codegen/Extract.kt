package dev.fragmentrpc.codegen

import com.google.gson.Gson
import org.jetbrains.kotlin.cli.jvm.compiler.EnvironmentConfigFiles
import org.jetbrains.kotlin.cli.jvm.compiler.KotlinCoreEnvironment
import org.jetbrains.kotlin.com.intellij.openapi.util.Disposer
import org.jetbrains.kotlin.config.CompilerConfiguration
import org.jetbrains.kotlin.psi.*
import java.io.File

data class NativeDeclaration(val language: String = "kotlin", val declaration: String, val kind: String, val packageName: String)
data class Document(val name: String, val body: String, val line: Int, val column: Int, val native: NativeDeclaration)

fun main(args: Array<String>) {
    val disposable = Disposer.newDisposable()
    try {
        val environment = KotlinCoreEnvironment.createForProduction(disposable, CompilerConfiguration(), EnvironmentConfigFiles.JVM_CONFIG_FILES)
        val factory = KtPsiFactory(environment.project, false)
        val documents = mutableListOf<Document>()
        for (path in args) {
            val source = File(path).readText()
            val file = factory.createFile(File(path).name, source)
            file.accept(object : KtTreeVisitorVoid() {
                override fun visitClassOrObject(declaration: KtClassOrObject) {
                    for (annotation in declaration.annotationEntries) {
                        val marker = annotation.shortName?.asString()
                        if (marker != "GraphQLFragment" && marker != "GraphQLQuery") continue
                        fun invalid(message: String): Nothing {
                            val line = source.take(annotation.textOffset).count { it == '\n' } + 1
                            error("$path:$line: error: $message")
                        }
                        val arguments = annotation.valueArguments
                        val literal = arguments.singleOrNull()?.getArgumentExpression() as? KtStringTemplateExpression
                            ?: invalid("GraphQL declarations require one literal string")
                        if (!literal.text.startsWith("\"\"\"")) invalid("Use a multiline triple-quoted GraphQL literal")
                        val body = literal.entries.joinToString("") { entry ->
                            when (entry) {
                                is KtLiteralStringTemplateEntry -> entry.text
                                // Kotlin's raw strings still interpolate '$'; this is the constant escape for GraphQL variables.
                                is KtBlockStringTemplateEntry -> if (entry.expression?.text == "'$'") "$" else invalid("Dynamic GraphQL interpolation is not supported")
                                else -> invalid("Dynamic GraphQL interpolation is not supported")
                            }
                        }
                        val start = literal.textOffset + 3
                        val prefix = source.take(start)
                        documents += Document(path, body, prefix.count { it == '\n' } + 1,
                            start - prefix.lastIndexOf('\n'),
                            NativeDeclaration(declaration = declaration.name ?: invalid("Name the GraphQL declaration"),
                                kind = if (marker == "GraphQLFragment") "fragment" else "query", packageName = file.packageFqName.asString()))
                    }
                    super.visitClassOrObject(declaration)
                }
            })
        }
        print(Gson().toJson(documents))
    } catch (error: Exception) {
        System.err.println(error.message)
        kotlin.system.exitProcess(1)
    } finally {
        Disposer.dispose(disposable)
    }
}
