import assert from "node:assert/strict";
import { createClient } from "@connectrpc/connect";
import { createGrpcTransport } from "@connectrpc/connect-node";
import { toJson } from "@bufbuild/protobuf";
import { AppService, GetUserPageResponseSchema, GetUserSummaryResponseSchema } from "../generated/app_pb.js";

const client = createClient(AppService, createGrpcTransport({ baseUrl: process.env.GRPC_URL ?? "http://127.0.0.1:50051" }));
const page = await client.getUserPage({ id: "u1" }, { timeoutMs: 5000 });
assert.equal(page.user?.organization?.name, "Northstar Studio");
assert.equal(Object.hasOwn(page.user!, "email"), false);
console.log("GetUserPage over gRPC:", JSON.stringify(toJson(GetUserPageResponseSchema, page), null, 2));
const summary = await client.getUserSummary({ id: "u2" }, { timeoutMs: 5000 });
assert.equal(summary.user?.name, "Ren Sato");
assert.equal(Object.hasOwn(summary.user!, "organization"), false);
console.log("GetUserSummary over gRPC:", JSON.stringify(toJson(GetUserSummaryResponseSchema, summary), null, 2));
console.log("PASS: two generated RPCs, projected responses, federation join.");
