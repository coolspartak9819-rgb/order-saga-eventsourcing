<?php
declare(strict_types=1);

function envValue(string $key, string $fallback): string
{
    $value = getenv($key);
    return $value === false || $value === '' ? $fallback : $value;
}

$pdo = new PDO(
    sprintf('mysql:host=%s;dbname=%s;charset=utf8mb4', envValue('DB_HOST', 'db'), envValue('DB_NAME', 'platform')),
    envValue('DB_USER', 'platform'),
    envValue('DB_PASSWORD', 'platform'),
    [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]
);

$producer = new RdKafka\Producer();
$producer->addBrokers(envValue('KAFKA_BROKERS', 'kafka:9092'));
$topic = $producer->newTopic('items.events');

while (true) {
    $pdo->beginTransaction();
    $row = $pdo->query(
        "SELECT id, aggregate_id, event_type, payload FROM outbox
         WHERE status = 'PENDING' ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED"
    )->fetch();

    if (!$row) {
        $pdo->commit();
        usleep(500000);
        continue;
    }

    try {
        $topic->produce(RD_KAFKA_PARTITION_UA, 0, json_encode([
            'id' => (int)$row['id'],
            'aggregate_id' => (int)$row['aggregate_id'],
            'event_type' => $row['event_type'],
            'payload' => json_decode($row['payload'], true, 512, JSON_THROW_ON_ERROR),
        ], JSON_THROW_ON_ERROR));
        $producer->flush(5000);
        $update = $pdo->prepare("UPDATE outbox SET status = 'PUBLISHED', published_at = NOW() WHERE id = ?");
        $update->execute([$row['id']]);
        $pdo->commit();
    } catch (Throwable $error) {
        $pdo->rollBack();
        error_log('outbox publish failed: ' . $error->getMessage());
        sleep(2);
    }
}
