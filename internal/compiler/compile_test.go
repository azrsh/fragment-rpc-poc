package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func compileTest(t *testing.T, sdl, body string, lock *ContractLock) *Output {
	t.Helper()
	out, err := Compile(sdl, []SourceFile{{Name: "test.graphql", Body: body}}, lock)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func mustContain(t *testing.T, text, part string) {
	t.Helper()
	if !strings.Contains(text, part) {
		t.Fatalf("missing %q in %s", part, text)
	}
}
func schema(t *testing.T) string { return read(t, filepath.Join(repoRoot(t), "schema.graphql")) }

func TestColocatedContracts(t *testing.T) {
	root := repoRoot(t)
	sources, err := CollectSources(root, []string{"app"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Compile(schema(t), sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Operations) != 2 || out.Operations[0].Name != "GetUserPage" || out.Operations[1].Name != "GetUserSummary" {
		t.Fatalf("operations: %+v", out.Operations)
	}
	fields := out.Operations[0].Fields[0].Fields
	names := []string{}
	for _, f := range fields {
		names = append(names, f.Name)
	}
	if !reflect.DeepEqual(names, []string{"avatarUrl", "id", "name", "organization"}) {
		t.Fatal(names)
	}
	if strings.Contains(out.Proto, "email") || strings.Contains(out.Proto, "organizationId") {
		t.Fatal("unselected backend fields leaked")
	}
	mustContain(t, out.Proto, "rpc GetUserPage(GetUserPageRequest) returns (GetUserPageResponse)")
	mustContain(t, out.FragmentTypes, "export type Avatar_user = { avatarUrl: string;")
	slices.Reverse(sources)
	reversed, err := Compile(schema(t), sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, reversed) {
		t.Fatal("source order changed the contract")
	}
}

func TestMergeAndAliases(t *testing.T) {
	out := compileTest(t, schema(t), `fragment Base on User { name organization { name } }
fragment More on User { name organization { website } }
query GetUser($id:ID!){profile:user(id:$id){...Base ...More label:name}}`, nil)
	u := out.Operations[0].Fields[0]
	if u.Name != "user" || u.ResponseName != "profile" || len(u.Fields) != 3 || u.Fields[0].ResponseName != "label" || u.Fields[0].Name != "name" || len(u.Fields[2].Fields) != 2 {
		t.Fatalf("merged: %+v", u)
	}
}

func TestInvalidDocuments(t *testing.T) {
	for _, test := range []struct{ body, match string }{
		{`query GetUser{user(id:"u1"){typo}}`, `Cannot query field "typo"`},
		{`query GetUser{user(id:"u1"){...Missing}}`, `Unknown fragment`},
		{`fragment A on User{...B} fragment B on User{...A} query GetUser{user(id:"u1"){...A}}`, `Cannot spread fragment`},
		{`query GetUser{user(id:"u1"){label:name label:email}}`, `conflict`},
		{`query GetUser{user(id:$id){name}}`, `is not defined`},
		{`query GetUser($id:ID!,$show:Boolean!){user(id:$id){name @include(if:$show)}}`, `Directive @include`},
		{`query GetUser($id:ID!="u1"){user(id:$id){name}}`, `Variable defaults`},
		{`{user(id:"u1"){name}}`, `Only named queries`},
		{`query GetUser{user(id:"u1"){full_name:name}}`, `lowerCamelCase`},
		{`query GetUser{kind:__typename}`, `Introspection field`},
	} {
		t.Run(test.match, func(t *testing.T) {
			_, err := Compile(schema(t), []SourceFile{{Name: "test.graphql", Body: test.body}}, nil)
			if err == nil {
				t.Fatal("accepted unsupported query")
			}
			mustContain(t, err.Error(), test.match)
		})
	}
	for _, sdl := range []string{`type Query{users:[User]} type User{name:String!}`, `type Query{users:[[User!]!]!} type User{name:String!}`} {
		_, err := Compile(sdl, []SourceFile{{Name: "test.graphql", Body: `query List{users{name}}`}}, nil)
		if err == nil {
			t.Fatal("accepted unsupported list")
		}
		mustContain(t, err.Error(), "Only non-null lists")
	}
	_, err := Compile(`type Query{user(id:ID="u1"):String}`, []SourceFile{{Name: "test.graphql", Body: `query GetUser{user}`}}, nil)
	if err == nil {
		t.Fatal("accepted schema argument default")
	}
	mustContain(t, err.Error(), "Schema argument defaults")
}

func TestPresenceAndLists(t *testing.T) {
	out := compileTest(t, `type Query{users(enabled:Boolean!):[User!]!} type User{name:String}`, `query List($enabled:Boolean!){users(enabled:$enabled){name}}`, nil)
	for _, part := range []string{"optional bool enabled", "repeated ListResponse_Users users", "optional string name"} {
		mustContain(t, out.Proto, part)
	}
}

func TestContractEvolution(t *testing.T) {
	initial := compileTest(t, schema(t), `query GetUser{user(id:"u1"){id name}}`, nil)
	changed := compileTest(t, schema(t), `query GetUser{user(id:"u1"){email id name}}`, &initial.Lock)
	before := initial.Lock.Messages["GetUserResponse_User"].Fields
	after := changed.Lock.Messages["GetUserResponse_User"].Fields
	if before["id"] != after["id"] || before["name"] != after["name"] || after["email"].Number != 3 {
		t.Fatal("renumbered existing fields")
	}
	if _, ok := before["email"]; ok {
		t.Fatal("mutated previous lock")
	}
	removed := compileTest(t, schema(t), `query GetUser{user(id:"u1"){id name}}`, &changed.Lock)
	mustContain(t, removed.Proto, "reserved 3;")
	mustContain(t, removed.Proto, `reserved "email";`)
	_, err := Compile(schema(t), []SourceFile{{Name: "test", Body: `query GetUser{user(id:"u1"){email id name}}`}}, &removed.Lock)
	if err == nil {
		t.Fatal("reused retired field")
	}
	_, err = Compile(strings.ReplaceAll(schema(t), "name: String!", "name: Int!"), []SourceFile{{Name: "test", Body: `query GetUser{user(id:"u1"){id name}}`}}, &initial.Lock)
	if err == nil {
		t.Fatal("accepted breaking type")
	}
	mustContain(t, err.Error(), "Breaking type change")
	initial.Lock.Messages["GetUserResponse_User"].Fields["name"] = LockedField{Number: 18999, Signature: "String!"}
	reserved := compileTest(t, schema(t), `query GetUser{user(id:"u1"){email id name}}`, &initial.Lock)
	if reserved.Lock.Messages["GetUserResponse_User"].Fields["email"].Number != 20000 {
		t.Fatal("used protobuf reserved field number")
	}
}

func TestNativeContracts(t *testing.T) {
	root := repoRoot(t)
	sources, err := CollectSources(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Compile(schema(t), sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, o := range out.Operations {
		names = append(names, o.Name)
	}
	if !reflect.DeepEqual(names, []string{"GetAndroidUserPage", "GetAndroidUserSummary", "GetIosUserPage", "GetIosUserSummary", "GetUserPage", "GetUserSummary"}) {
		t.Fatal(names)
	}
	for _, platform := range []string{"ios", "android"} {
		changedSources := slices.Clone(sources)
		for i, s := range changedSources {
			if strings.HasPrefix(s.Name, "mobile/"+platform+"/") && (strings.HasSuffix(s.Name, "UserCard.swift") || strings.HasSuffix(s.Name, "UserCard.kt")) {
				changedSources[i].Body = strings.Replace(s.Body, "  name\n", "  name\n  email\n", 1)
			}
		}
		changed, err := Compile(schema(t), changedSources, &out.Lock)
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range changed.Operations {
			email := false
			for _, f := range o.Fields[0].Fields {
				email = email || f.Name == "email"
			}
			expected := "GetIosUserPage"
			if platform == "android" {
				expected = "GetAndroidUserPage"
			}
			if email != (o.Name == expected) {
				t.Fatalf("%s changed %s", platform, o.Name)
			}
		}
	}
	native, err := NativeModels(out, sources)
	if err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []string{"Ios", "Android"} {
		found := false
		for _, m := range native.Models {
			if m.Name == prefix+"AvatarFragmentData" {
				found = true
				if len(m.Fields) != 1 || m.Fields[0].Name != "avatarUrl" {
					t.Fatal(m)
				}
			}
			if m.Name == prefix+"UserCardFragmentData" {
				for _, f := range m.Fields {
					if f.Name == "organization" && !f.Optional {
						t.Fatal("lost optional relation")
					}
				}
			}
		}
		if !found {
			t.Fatal("missing avatar DTO")
		}
		conversion := false
		for _, c := range native.Conversions {
			if c.To.Name == prefix+"AvatarFragmentData" {
				if strings.Contains(c.From.Name, "Summary") {
					t.Fatal("summary cannot supply avatar")
				}
				conversion = conversion || c.From.Name == prefix+"UserCardFragmentData"
			}
		}
		if !conversion {
			t.Fatal("missing parent-to-child conversion")
		}
	}
	mustContain(t, SwiftModels(native), "self.hasOrganization ?")
	var lock ContractLock
	if err = json.Unmarshal([]byte(read(t, filepath.Join(root, "generated/schema.lock.json"))), &lock); err != nil {
		t.Fatal(err)
	}
	locked, err := Compile(schema(t), sources, &lock)
	if err != nil {
		t.Fatal(err)
	}
	if locked.Proto != read(t, filepath.Join(root, "generated/app.proto")) {
		t.Fatal("stale generated contract")
	}
	if SwiftModels(native) != read(t, filepath.Join(root, "generated/NativeFragments.swift")) {
		t.Fatal("stale Swift models")
	}
}

func TestNativeExtractionLocations(t *testing.T) {
	dir := t.TempDir()
	root := repoRoot(t)
	swift := "// @GraphQLFragment(\"fake\") enum Fake {}\n@GraphQLFragment(\"\"\"\n    fragment Avatar_user on User {\n      avatarUrl\n    }\n    \"\"\")\nenum AvatarFragment {}\n"
	kotlin := "package example\n// @GraphQLQuery(\"fake\") object Fake\n@GraphQLQuery(\"\"\"\nquery Page(${'$'}id: ID!) { user(id: ${'$'}id) { ...Avatar_user } }\n\"\"\")\nobject PageQuery\n"
	for name, body := range map[string]string{"Avatar.swift": swift, "Page.kt": kotlin} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	sources, err := CollectSources(root, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 {
		t.Fatal(sources)
	}
	for _, s := range sources {
		if s.Native.Language == "swift" && s.Line != 3 {
			t.Fatal(s)
		}
		if s.Native.Language == "kotlin" {
			mustContain(t, s.Body, "query Page($id: ID!)")
		}
	}
	sdl := `type Query{user(id:ID!):User} type User{avatarUrl:String!}`
	out, err := Compile(sdl, sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Operations[0].Name != "Page" {
		t.Fatal("missing native operation")
	}
	for i := range sources {
		sources[i].Body = strings.ReplaceAll(sources[i].Body, "avatarUrl", "unknownField")
	}
	_, err = Compile(sdl, sources, nil)
	if err == nil {
		t.Fatal("accepted invalid field")
	}
	mustContain(t, err.Error(), "Avatar.swift:4:7")
}

func TestNativeExtractionRejectsInterpolation(t *testing.T) {
	for ext, body := range map[string]string{"swift": "@GraphQLFragment(\"\"\"\nfragment F on User { \\(field) }\n\"\"\")\nenum F {}", "kt": "@GraphQLFragment(\"\"\"\nfragment F on User { $field }\n\"\"\")\nobject F"} {
		t.Run(ext, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "Dynamic."+ext), []byte(body), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := CollectSources(repoRoot(t), []string{dir})
			if err == nil {
				t.Fatal("accepted interpolation")
			}
			mustContain(t, err.Error(), "interpolation")
		})
	}
}
