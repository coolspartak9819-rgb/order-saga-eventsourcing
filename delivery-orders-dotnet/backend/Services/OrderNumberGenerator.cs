using System.Security.Cryptography;

namespace DeliveryOrders.Api.Services;

public interface IOrderNumberGenerator
{
    string Generate(DateTimeOffset now);
}

public sealed class OrderNumberGenerator : IOrderNumberGenerator
{
    public string Generate(DateTimeOffset now)
    {
        Span<byte> bytes = stackalloc byte[4];
        RandomNumberGenerator.Fill(bytes);
        var code = Convert.ToHexString(bytes)[..6];
        return $"DLV-{now:yyyy}-{code}";
    }
}
