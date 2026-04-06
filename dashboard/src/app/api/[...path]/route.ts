import { NextRequest } from "next/server";

const API_URL = process.env.API_URL || "http://localhost:8080";

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  return proxy(request, await params);
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  return proxy(request, await params);
}

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  return proxy(request, await params);
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  return proxy(request, await params);
}

async function proxy(request: NextRequest, params: { path: string[] }) {
  const path = params.path.join("/");
  const url = `${API_URL}/api/${path}`;

  const headers = new Headers();
  const authHeader = request.headers.get("Authorization");
  if (authHeader) {
    headers.set("Authorization", authHeader);
  }
  const contentType = request.headers.get("Content-Type");
  if (contentType) {
    headers.set("Content-Type", contentType);
  }

  const res = await fetch(url, {
    method: request.method,
    headers,
    body: request.method !== "GET" ? await request.text() : undefined,
  });

  return new Response(res.body, {
    status: res.status,
    headers: {
      "Content-Type": res.headers.get("Content-Type") || "application/json",
    },
  });
}
