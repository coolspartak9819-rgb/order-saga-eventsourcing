using DeliveryOrders.Api.Persistence;
using DeliveryOrders.Api.Models;
using DeliveryOrders.Api.Services;
using Microsoft.AspNetCore.Diagnostics;
using Microsoft.EntityFrameworkCore;
using System.Text.Json.Serialization;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddProblemDetails();
builder.Services.AddOpenApi();
builder.Services.ConfigureHttpJsonOptions(options =>
    options.SerializerOptions.Converters.Add(new JsonStringEnumConverter()));
builder.Services.AddHealthChecks().AddDbContextCheck<DeliveryOrdersDbContext>();
builder.Services.AddDbContext<DeliveryOrdersDbContext>(options =>
    options.UseSqlite(builder.Configuration.GetConnectionString("DefaultConnection")));
builder.Services.AddSingleton<IOrderNumberGenerator, OrderNumberGenerator>();
builder.Services.AddCors(options => options.AddPolicy("frontend", policy =>
    policy.AllowAnyOrigin().AllowAnyHeader().AllowAnyMethod()));

var app = builder.Build();

app.UseExceptionHandler(exceptionApp => exceptionApp.Run(async context =>
{
    var error = context.Features.Get<IExceptionHandlerFeature>()?.Error;
    var problem = Results.Problem(
        statusCode: StatusCodes.Status500InternalServerError,
        title: "Unexpected server error",
        detail: app.Environment.IsDevelopment() ? error?.Message : null);
    await problem.ExecuteAsync(context);
}));

app.UseCors("frontend");
app.MapOpenApi();
app.MapHealthChecks("/health");

using (var scope = app.Services.CreateScope())
{
    var database = scope.ServiceProvider.GetRequiredService<DeliveryOrdersDbContext>();
    await database.Database.EnsureCreatedAsync();
}

var orders = app.MapGroup("/api/orders").WithTags("Orders");

orders.MapGet("", async (DeliveryOrdersDbContext db, string? search, OrderStatus? status, CancellationToken cancellationToken) =>
{
    var query = db.Orders.AsNoTracking().AsQueryable();
    if (!string.IsNullOrWhiteSpace(search))
    {
        var value = search.Trim();
        query = query.Where(order => order.OrderNumber.Contains(value) ||
            order.SenderCity.Contains(value) || order.RecipientCity.Contains(value));
    }
    if (status.HasValue) query = query.Where(order => order.Status == status.Value);

    var result = (await query.Select(order => new OrderListItem(order.Id, order.OrderNumber, order.SenderCity,
            order.RecipientCity, order.WeightKg, order.PickupDate, order.Status, order.CreatedAt))
        .ToListAsync(cancellationToken)).OrderByDescending(order => order.CreatedAt).ToList();
    return Results.Ok(result);
});

orders.MapGet("/{id:guid}", async (Guid id, DeliveryOrdersDbContext db, CancellationToken cancellationToken) =>
{
    var order = await db.Orders.AsNoTracking().FirstOrDefaultAsync(item => item.Id == id, cancellationToken);
    return order is null
        ? Results.NotFound(new { message = "Order not found" })
        : Results.Ok(ToDetails(order));
});

orders.MapPost("", async (CreateOrderRequest request, DeliveryOrdersDbContext db,
    IOrderNumberGenerator numberGenerator, CancellationToken cancellationToken) =>
{
    var errors = Validate(request);
    if (errors.Count > 0) return Results.ValidationProblem(errors);

    var now = DateTimeOffset.UtcNow;
    var order = new DeliveryOrder
    {
        Id = Guid.NewGuid(),
        OrderNumber = numberGenerator.Generate(now),
        SenderCity = request.SenderCity!.Trim(),
        SenderAddress = request.SenderAddress!.Trim(),
        RecipientCity = request.RecipientCity!.Trim(),
        RecipientAddress = request.RecipientAddress!.Trim(),
        WeightKg = request.WeightKg,
        PickupDate = request.PickupDate,
        CreatedAt = now,
        Status = OrderStatus.Created
    };

    db.Orders.Add(order);
    await db.SaveChangesAsync(cancellationToken);
    return Results.Created($"/api/orders/{order.Id}", ToDetails(order));
});

app.Run();

static Dictionary<string, string[]> Validate(CreateOrderRequest request)
{
    var errors = new Dictionary<string, string[]>();
    AddIfEmpty(errors, nameof(request.SenderCity), request.SenderCity, "Укажите город отправителя.");
    AddIfEmpty(errors, nameof(request.SenderAddress), request.SenderAddress, "Укажите адрес отправителя.");
    AddIfEmpty(errors, nameof(request.RecipientCity), request.RecipientCity, "Укажите город получателя.");
    AddIfEmpty(errors, nameof(request.RecipientAddress), request.RecipientAddress, "Укажите адрес получателя.");
    if (request.WeightKg <= 0) errors[nameof(request.WeightKg)] = ["Вес должен быть больше нуля."];
    if (request.WeightKg > 10_000) errors[nameof(request.WeightKg)] = ["Вес не может превышать 10 000 кг."];
    if (request.PickupDate < DateOnly.FromDateTime(DateTime.UtcNow.Date)) errors[nameof(request.PickupDate)] = ["Дата забора не может быть в прошлом."];
    return errors;
}

static void AddIfEmpty(Dictionary<string, string[]> errors, string key, string? value, string message)
{
    if (string.IsNullOrWhiteSpace(value)) errors[key] = [message];
}

static OrderDetails ToDetails(DeliveryOrder order) => new(order.Id, order.OrderNumber, order.SenderCity,
    order.SenderAddress, order.RecipientCity, order.RecipientAddress, order.WeightKg, order.PickupDate,
    order.Status, order.CreatedAt);

public partial class Program { }
