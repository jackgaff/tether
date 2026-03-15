import { Info } from "lucide-react";

const routes = [
  { method: "GET", path: "/", description: "Root handler — confirms the API is reachable." },
  { method: "GET", path: "/openapi.yaml", description: "Serves the OpenAPI spec for this API." },
  { method: "GET", path: "/health", description: "Returns service health, environment, auth mode, and DB config status." },
  { method: "GET", path: "/api/v1/check-ins", description: "Lists all check-in records. Returns an array ordered by most recent." },
  { method: "POST", path: "/api/v1/check-ins", description: "Creates a new check-in record. Expects JSON body with patientId, summary, status, and agent." },
];

const methodColor: Record<string, { bg: string; text: string }> = {
  GET: { bg: "bg-gray-100", text: "text-gray-700" },
  POST: { bg: "bg-green-50", text: "text-green-700" },
  PUT: { bg: "bg-amber-50", text: "text-amber-700" },
  DELETE: { bg: "bg-red-50", text: "text-red-700" },
};

export function ApiSurface() {
  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">API Surface</h1>
        <p className="mt-0.5 text-sm text-gray-400">
          Implemented routes served by the Go API at{" "}
          <code className="font-mono text-xs bg-gray-100 px-1.5 py-0.5 rounded text-gray-600">
            localhost:8080
          </code>
        </p>
      </div>

      <div className="flex items-start gap-3 bg-gray-50 border border-gray-200 rounded-xl px-4 py-3.5 mb-5">
        <Info size={15} className="text-gray-400 mt-0.5 flex-shrink-0" strokeWidth={2} />
        <p className="text-sm text-gray-600">
          The Go API is not currently running. Start it with{" "}
          <code className="font-mono text-xs bg-gray-100 px-1 rounded">bun run dev:api</code> to
          connect the frontend to live data.
        </p>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl overflow-hidden mb-5">
        <div className="px-5 py-4 border-b border-gray-100">
          <h2 className="text-sm font-semibold text-gray-900">Routes</h2>
        </div>
        <div className="divide-y divide-gray-50">
          {routes.map((route) => {
            const color = methodColor[route.method] ?? { bg: "bg-gray-100", text: "text-gray-600" };
            return (
              <div key={`${route.method}-${route.path}`} className="flex items-start gap-4 px-5 py-4">
                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold font-mono flex-shrink-0 ${color.bg} ${color.text}`}>
                  {route.method}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="font-mono text-sm text-gray-900">{route.path}</p>
                  <p className="mt-0.5 text-xs text-gray-500">{route.description}</p>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100">
          <h2 className="text-sm font-semibold text-gray-900">Example Requests</h2>
        </div>
        <div className="px-5 py-4 space-y-4">
          <div>
            <p className="text-xs font-medium text-gray-500 mb-2">Health check</p>
            <pre className="bg-gray-950 text-gray-200 text-xs rounded-lg px-4 py-3 overflow-x-auto">
              {`curl http://localhost:8080/health`}
            </pre>
          </div>
          <div>
            <p className="text-xs font-medium text-gray-500 mb-2">Create a check-in</p>
            <pre className="bg-gray-950 text-gray-200 text-xs rounded-lg px-4 py-3 overflow-x-auto">
              {`curl -X POST http://localhost:8080/api/v1/check-ins \\
  -H "Content-Type: application/json" \\
  -d '{"patientId":"patient-001","summary":"...","status":"completed","agent":"call-agent"}'`}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
}
