import type { APIRoute } from "astro";

export const GET: APIRoute = async ({ url, request }) => {
  const urlFromUrl = Object.fromEntries(url.searchParams.entries());
  const urlFromRequest = new URL(request.url);
  const paramsFromRequest = Object.fromEntries(
    urlFromRequest.searchParams.entries(),
  );

  return new Response(
    JSON.stringify({
      message: "API is working",
      url_params: urlFromUrl,
      request_params: paramsFromRequest,
      url_href: url.href,
      request_url: request.url,
    }),
    {
      status: 200,
      headers: { "Content-Type": "application/json" },
    },
  );
};
