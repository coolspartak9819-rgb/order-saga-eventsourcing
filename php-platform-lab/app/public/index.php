<?php
declare(strict_types=1);

header('Content-Type: application/json; charset=utf-8');

final class RedisClient
{
    private $socket;

    public function __construct(string $host, int $port)
    {
        $this->socket = @fsockopen($host, $port, $errno, $error, 1.0);
        if (!$this->socket) {
            throw new RuntimeException("Redis connection failed: {$error}");
        }
        stream_set_timeout($this->socket, 1);
    }

    public function command(string ...$parts): mixed
    {
        $request = '*' . count($parts) . "\r\n";
        foreach ($parts as $part) {
            $request .= '$' . strlen($part) . "\r\n{$part}\r\n";
        }
        fwrite($this->socket, $request);
        $line = fgets($this->socket);
        if ($line === false) {
            throw new RuntimeException('Redis read failed');
        }
        return match ($line[0]) {
            '+' => trim(substr($line, 1)),
            ':' => (int)trim(substr($line, 1)),
            '$' => $this->readBulk((int)trim(substr($line, 1))),
            '-' => throw new RuntimeException(trim(substr($line, 1))),
            default => throw new RuntimeException('Unexpected Redis response'),
        };
    }

    private function readBulk(int $length): ?string
    {
        if ($length < 0) {
            return null;
        }
        $value = stream_get_contents($this->socket, $length + 2);
        return substr($value === false ? '' : $value, 0, $length);
    }
}

function envValue(string $key, string $fallback): string
{
    $value = getenv($key);
    return $value === false || $value === '' ? $fallback : $value;
}

function database(): PDO
{
    static $pdo;
    if (!$pdo instanceof PDO) {
        $pdo = new PDO(
            sprintf('mysql:host=%s;dbname=%s;charset=utf8mb4', envValue('DB_HOST', 'db'), envValue('DB_NAME', 'platform')),
            envValue('DB_USER', 'platform'),
            envValue('DB_PASSWORD', 'platform'),
            [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]
        );
    }
    return $pdo;
}

function redis(): RedisClient
{
    static $client;
    return $client ??= new RedisClient(envValue('REDIS_HOST', 'redis'), (int)envValue('REDIS_PORT', '6379'));
}

function jsonResponse(array $body, int $status = 200): never
{
    http_response_code($status);
    echo json_encode($body, JSON_UNESCAPED_SLASHES);
    exit;
}

$started = microtime(true);
$path = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?: '/';
$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';

try {
    if ($path === '/health') {
        jsonResponse(['status' => 'ok']);
    }

    if ($path === '/ready') {
        database()->query('SELECT 1');
        redis()->command('PING');
        jsonResponse(['status' => 'ready']);
    }

    if ($path === '/metrics') {
        $elapsed = (microtime(true) - $started) * 1000;
        header('Content-Type: text/plain; version=0.0.4');
        echo "php_platform_requests_total 1\n";
        echo 'php_platform_request_duration_ms ' . round($elapsed, 2) . "\n";
        exit;
    }

    if ($path === '/items' && $method === 'GET') {
        $items = database()->query('SELECT id, title, price FROM items ORDER BY id DESC')->fetchAll();
        jsonResponse(['items' => $items]);
    }

    if ($path === '/items' && $method === 'POST') {
        $idempotencyKey = $_SERVER['HTTP_IDEMPOTENCY_KEY'] ?? '';
        if ($idempotencyKey !== '') {
            $cachedResponse = database()->prepare(
                'SELECT response_body, status_code FROM idempotency_keys WHERE idempotency_key = ?'
            );
            $cachedResponse->execute([$idempotencyKey]);
            $stored = $cachedResponse->fetch();
            if ($stored) {
                jsonResponse(json_decode($stored['response_body'], true, 512, JSON_THROW_ON_ERROR), (int)$stored['status_code']);
            }
        }

        $input = json_decode(file_get_contents('php://input'), true, 512, JSON_THROW_ON_ERROR);
        if (!is_array($input) || !isset($input['title'], $input['price'])) {
            jsonResponse(['error' => 'title and price are required'], 400);
        }
        $pdo = database();
        $pdo->beginTransaction();
        try {
            $stmt = $pdo->prepare('INSERT INTO items (title, price) VALUES (?, ?)');
            $stmt->execute([(string)$input['title'], (int)$input['price']]);
            $id = (int)$pdo->lastInsertId();
            $event = json_encode([
                'item_id' => $id,
                'title' => (string)$input['title'],
                'price' => (int)$input['price'],
            ], JSON_THROW_ON_ERROR);
            $outbox = $pdo->prepare(
                'INSERT INTO outbox (aggregate_id, event_type, payload) VALUES (?, ?, ?)'
            );
            $outbox->execute([$id, 'ItemCreated', $event]);
            if ($idempotencyKey !== '') {
                $response = json_encode(['id' => $id], JSON_THROW_ON_ERROR);
                $idempotency = $pdo->prepare(
                    'INSERT INTO idempotency_keys (idempotency_key, response_body, status_code) VALUES (?, ?, ?)'
                );
                $idempotency->execute([$idempotencyKey, $response, 201]);
            }
            $pdo->commit();
            jsonResponse(['id' => $id], 201);
        } catch (Throwable $error) {
            $pdo->rollBack();
            throw $error;
        }
    }

    jsonResponse(['error' => 'not found'], 404);
} catch (Throwable $error) {
    jsonResponse(['error' => $error->getMessage()], 503);
}
