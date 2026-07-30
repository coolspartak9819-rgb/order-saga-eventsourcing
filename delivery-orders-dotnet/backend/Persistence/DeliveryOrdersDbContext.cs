using DeliveryOrders.Api.Models;
using Microsoft.EntityFrameworkCore;

namespace DeliveryOrders.Api.Persistence;

public sealed class DeliveryOrdersDbContext(DbContextOptions<DeliveryOrdersDbContext> options) : DbContext(options)
{
    public DbSet<DeliveryOrder> Orders => Set<DeliveryOrder>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        var order = modelBuilder.Entity<DeliveryOrder>();
        order.HasKey(item => item.Id);
        order.HasIndex(item => item.OrderNumber).IsUnique();
        order.Property(item => item.OrderNumber).HasMaxLength(32).IsRequired();
        order.Property(item => item.SenderCity).HasMaxLength(120).IsRequired();
        order.Property(item => item.SenderAddress).HasMaxLength(240).IsRequired();
        order.Property(item => item.RecipientCity).HasMaxLength(120).IsRequired();
        order.Property(item => item.RecipientAddress).HasMaxLength(240).IsRequired();
        order.Property(item => item.WeightKg).HasPrecision(10, 2);
        order.Property(item => item.Status).HasConversion<string>().HasMaxLength(32);
        order.Property(item => item.PickupDate).HasConversion<string>();
    }
}
