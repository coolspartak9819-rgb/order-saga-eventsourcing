namespace DeliveryOrders.Api.Models;

public sealed class DeliveryOrder
{
    public Guid Id { get; set; }
    public string OrderNumber { get; set; } = string.Empty;
    public string SenderCity { get; set; } = string.Empty;
    public string SenderAddress { get; set; } = string.Empty;
    public string RecipientCity { get; set; } = string.Empty;
    public string RecipientAddress { get; set; } = string.Empty;
    public decimal WeightKg { get; set; }
    public DateOnly PickupDate { get; set; }
    public OrderStatus Status { get; set; } = OrderStatus.Created;
    public DateTimeOffset CreatedAt { get; set; }
}

public enum OrderStatus
{
    Created = 0,
    Accepted = 1,
    InTransit = 2,
    Delivered = 3,
    Cancelled = 4
}
