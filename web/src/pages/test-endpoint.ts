import type { APIRoute } from "astro";

export const GET: APIRoute = async (context) => {
  // Try to manually parse from various sources
  const manualParams: Record<string, string> = {};

  // Check if the request URL has query string
  if (context.request?.url) {
    const fullUrl = context.request.url;
    const queryIndex = fullUrl.indexOf("?");
    if (queryIndex > -1) {
      const queryString = fullUrl.substring(queryIndex + 1);
      const pairs = queryString.split("&");
      for (const pair of pairs) {
        const [key, value] = pair.split("=");
        const k = decodeURIComponent(key || "");
        const v = decodeURIComponent(value || "");
        if (k) manualParams[k] = v;
      }
    }
  }

  return new Response(
    JSON.stringify({
      message: "Non-API endpoint test",
      context: {
        url: context.url?.href,
        request_url: context.request?.url,
        searchParams_entries: context.url
          ? Array.from(context.url.searchParams.entries())
          : [],
        manual_params: manualParams,
      },
    }),
    {
      status: 200,
      headers: { "Content-Type": "application/json" },
    },
  );
};
