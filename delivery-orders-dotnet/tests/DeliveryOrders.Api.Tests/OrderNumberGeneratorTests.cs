using DeliveryOrders.Api.Services;
using Xunit;

namespace DeliveryOrders.Api.Tests;

public sealed class OrderNumberGeneratorTests
{
    [Fact]
    public void Generates_a_readable_unique_order_number()
    {
        var generator = new OrderNumberGenerator();
        var now = new DateTimeOffset(2026, 7, 30, 10, 0, 0, TimeSpan.Zero);

        var first = generator.Generate(now);
        var second = generator.Generate(now);

        Assert.Matches("^DLV-2026-[0-9A-F]{6}$", first);
        Assert.NotEqual(first, second);
    }
}
