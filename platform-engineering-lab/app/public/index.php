<?php
declare(strict_types=1);

$path = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?: '/';
$version = getenv('APP_VERSION') ?: 'local';
$start = microtime(true);

function dependencyReachable(string $host, int $port): bool
{
    $socket = @fsockopen($host, $port, $errno, $error, 0.5);
    if (!$socket) {
        return false;
    }
    fclose($socket);
    return true;
}

if ($path === '/health') {
    header('Content-Type: application/json');
    echo json_encode(['status' => 'ok', 'version' => $version]);
    exit;
}

if ($path === '/ready') {
    $mysqlReady = dependencyReachable(getenv('DB_HOST') ?: 'mysql', 3306);
    $redisReady = dependencyReachable(getenv('REDIS_HOST') ?: 'redis', 6379);
    $ready = $mysqlReady && $redisReady;
    http_response_code($ready ? 200 : 503);
    header('Content-Type: application/json');
    echo json_encode(['status' => $ready ? 'ready' : 'not_ready', 'mysql' => $mysqlReady, 'redis' => $redisReady]);
    exit;
}

if ($path === '/metrics') {
    header('Content-Type: text/plain; version=0.0.4');
    echo "platform_app_up 1\n";
    echo 'platform_app_request_duration_seconds ' . (microtime(true) - $start) . "\n";
    exit;
}

if ($path === '/api/content') {
    if (getenv('FAIL_MODE') === 'true') {
        http_response_code(503);
        echo json_encode(['error' => 'injected failure']);
        exit;
    }
    header('Content-Type: application/json');
    header('Cache-Control: public, max-age=30, stale-while-revalidate=60');
    echo json_encode(['version' => $version, 'content' => 'platform-lab', 'generated_at' => date(DATE_ATOM)]);
    exit;
}

http_response_code(404);
header('Content-Type: application/json');
echo json_encode(['error' => 'not found']);
