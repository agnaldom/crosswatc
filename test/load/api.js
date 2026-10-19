// k6 load test for crosswatc-api (spec §53 SLOs).
//   BASE_URL=http://localhost:8080 SQUADS=20 k6 run test/load/api.js
// The API must run with CROSSWATC_AUTH_MODE=dev against data from `crosswatc-dbtool seed-load`.
import http from "k6/http";
import { check, sleep } from "k6";
import { Trend } from "k6/metrics";

const BASE = __ENV.BASE_URL || "http://localhost:8080";
const SQUADS = Number(__ENV.SQUADS || 20);

export const options = {
  scenarios: {
    browse: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "20s", target: Number(__ENV.VUS || 50) },
        { duration: __ENV.HOLD || "1m", target: Number(__ENV.VUS || 50) },
        { duration: "10s", target: 0 },
      ],
    },
  },
  thresholds: {
    // §53: GET /resources/{id} p95 < 400ms
    "http_req_duration{name:get_resource}": ["p(95)<400"],
    "http_req_duration{name:get_state}": ["p(95)<400"],
    "http_req_duration{name:list_resources}": ["p(95)<800"],
    "http_req_duration{name:list_events}": ["p(95)<800"],
    // §53: availability 99.5%
    http_req_failed: ["rate<0.005"],
    checks: ["rate>0.995"],
  },
};

const pageDepth = new Trend("pages_walked");

function headers(vu) {
  return { headers: { "X-Dev-User": `load-${vu}`, "X-Dev-Squads": `load${vu % SQUADS}` } };
}

export default function () {
  const h = headers(__VU);
  let res = http.get(`${BASE}/api/v1/resources?limit=50`, Object.assign({ tags: { name: "list_resources" } }, h));
  check(res, { "list 200": (r) => r.status === 200 });
  const page = res.status === 200 ? res.json() : { items: [] };
  if (page.nextCursor && Math.random() < 0.3) {
    res = http.get(`${BASE}/api/v1/resources?limit=50&cursor=${page.nextCursor}`, Object.assign({ tags: { name: "list_resources" } }, h));
    check(res, { "page 2 200": (r) => r.status === 200 });
    pageDepth.add(2);
  }
  if (!page.items.length) {
    sleep(1);
    return;
  }
  const id = page.items[Math.floor(Math.random() * page.items.length)].id;
  const r1 = http.get(`${BASE}/api/v1/resources/${id}`, Object.assign({ tags: { name: "get_resource" } }, h));
  const r2 = http.get(`${BASE}/api/v1/resources/${id}/state`, Object.assign({ tags: { name: "get_state" } }, h));
  const r3 = http.get(`${BASE}/api/v1/resources/${id}/events?limit=20`, Object.assign({ tags: { name: "list_events" } }, h));
  const r4 = http.get(`${BASE}/api/v1/resources/${id}/topology`, Object.assign({ tags: { name: "get_topology" } }, h));
  check(r1, { "resource 200": (r) => r.status === 200 });
  check(r2, { "state 200": (r) => r.status === 200 });
  check(r3, { "events 200": (r) => r.status === 200 });
  check(r4, { "topology 200": (r) => r.status === 200 });
  sleep(Math.random() * 2);
}
