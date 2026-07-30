namespace DeliveryOrders.Api.Models;

public sealed record CreateOrderRequest(
    string? SenderCity,
    string? SenderAddress,
    string? RecipientCity,
    string? RecipientAddress,
    decimal WeightKg,
    DateOnly PickupDate);

public sealed record OrderListItem(
    Guid Id,
    string OrderNumber,
    string SenderCity,
    string RecipientCity,
    decimal WeightKg,
    DateOnly PickupDate,
    OrderStatus Status,
    DateTimeOffset CreatedAt);

public sealed record OrderDetails(
    Guid Id,
    string OrderNumber,
    string SenderCity,
    string SenderAddress,
    string RecipientCity,
    string RecipientAddress,
    decimal WeightKg,
    DateOnly PickupDate,
    OrderStatus Status,
    DateTimeOffset CreatedAt);
