# Estimativa de custos (spec §60)

O preview (`POST /resources/{id}/changes/preview`) e o submit trazem `cost`:

```json
"cost": {
  "current": {"monthly": 73.0, "currency": "USD", "lines": [{"component": "vm", "sku": "Standard_D2as_v5", "region": "brazilsouth", "unitPrice": 0.1, "quantity": 1, "monthly": 73.0}]},
  "new":     {"monthly": 146.0, "currency": "USD", "lines": ["..."]},
  "monthlyDelta": 73.0,
  "currency": "USD",
  "partial": false
}
```

A variação também vai para o corpo do Pull Request.

## Habilitando para um tipo de recurso

Anotar o XRD com `crosswatc.io/cost` (YAML/JSON):

```yaml
metadata:
  annotations:
    crosswatc.io/cost: |
      - name: vm
        service: Virtual Machines           # serviceName da Azure Retail Prices API
        sku: spec.parameters.vmSize         # → armSkuName
        region: spec.parameters.location    # → armRegionName
        regionDefault: brazilsouth
        quantity: spec.parameters.count     # opcional (default 1)
        unit: hour                          # hour (× hoursPerMonth, default 730) | month | unit
      - name: disk
        service: Storage
        sku: spec.parameters.diskSku
        regionDefault: brazilsouth
        quantityPerUnit: spec.parameters.diskGb   # medidores por GB
        unit: month
```

Tipos sem a annotation não exibem custo.

## Fonte de preços

- `cost.provider: azure` (default) consulta `https://prices.azure.com/api/retail/prices` (pública, sem credenciais): `priceType = Consumption`, ignora Spot, Low Priority e Windows, escolhe o menor preço Linux on-demand; cache de 24 h por SKU/região.
- `cost.currency`: moeda da API (USD, BRL, EUR…).
- Componentes sem preço → `partial: true` e aviso; a mudança nunca é bloqueada por falta de estimativa.
- São preços de lista: não incluem descontos (EA/MCA), reservas, savings plans ou impostos.
