// Cloudflare Worker для проксі Go Playground
// Шлях: functions/api/run-code.js

export async function onRequest(context) {
    if (context.request.method === 'OPTIONS') {
        // Обробка CORS Pre-flight запиту
        return new Response(null, {
            headers: {
                'Access-Control-Allow-Origin': '*',
                'Access-Control-Allow-Methods': 'POST, OPTIONS',
                'Access-Control-Allow-Headers': 'Content-Type',
                'Access-Control-Max-Age': '86400',
            },
            status: 204 // No Content
        });
    }

    if (context.request.method !== 'POST') {
        return new Response('Дозволено лише POST-запити', { status: 405 });
    }

    try {
        const url = 'https://play.golang.org/p/compile';

        // Клонуємо запит для використання в fetch
        const proxyRequest = context.request.clone();

        const response = await fetch(url, {
            method: 'POST',
            headers: proxyRequest.headers,
            body: proxyRequest.body,
        });

        // Копіюємо відповідь
        const clonedResponse = new Response(response.body, response);

        // Додаємо CORS-заголовок
        clonedResponse.headers.set('Access-Control-Allow-Origin', '*');
        
        return clonedResponse;

    } catch (error) {
        return new Response(`Помилка проксі: ${error.message}`, { status: 500 });
    }
}