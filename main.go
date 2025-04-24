package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

type Order struct {
	ID         int
	Amount     float64
	MerchantID int
}

type Certificate struct {
	ID     int
	Amount float64
	Orders []Order
}

// generateOrders genera 612 órdenes para cada uno de los 3500 comerciantes
func generateOrders() ([]Order, error) {
	const numMerchants = 3500
	const ordersPerMerchant = 612
	totalOrders := numMerchants * ordersPerMerchant
	
	// Pre-asignar memoria para todas las órdenes mejora significativamente el rendimiento
	orders := make([]Order, 0, totalOrders)
	
	// Crear un generador de números aleatorios con semilla para reproducibilidad
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	
	orderID := 1
	
	// Para cada comerciante, generar sus órdenes
	for merchantID := 1; merchantID <= numMerchants; merchantID++ {
		for j := 0; j < ordersPerMerchant; j++ {
			// Generar un monto aleatorio entre 10.0 y 1000.0
			amount := 10.0 + r.Float64()*990.0
			
			// Redondear a 2 decimales
			amount = float64(int(amount*100)) / 100
			
			// Crear la orden y añadirla al slice
			order := Order{
				ID:         orderID,
				Amount:     amount,
				MerchantID: merchantID,
			}
			orders = append(orders, order)
			orderID++
		}
		
		// Mostrar progreso cada 100 comerciantes
		if merchantID%100 == 0 {
			fmt.Printf("Generadas %d órdenes para %d de %d comerciantes\n", 
				merchantID*ordersPerMerchant, merchantID, numMerchants)
		}
	}
	
	return orders, nil
}

// Función para generar certificados basados en un límite de monto
// Con optimización para llenar al máximo cada certificado, dejando solo los últimos 30 para equilibrarse
func generateCertificates(orders []Order, limitAmount float64) []Certificate {
	// Verificación adicional para asegurar que ningún certificado exceda el límite
	const ABSOLUTE_LIMIT = 500000.0
	if limitAmount > ABSOLUTE_LIMIT {
		limitAmount = ABSOLUTE_LIMIT
	}
	
	// Número aproximado de certificados objetivo basado en equilibrio de montos
	totalAmount := 0.0
	for _, order := range orders {
		totalAmount += order.Amount
	}
	
	// Calcular la cantidad estimada de certificados
	estimatedNumCertificates := int(math.Ceil(totalAmount / limitAmount))
	reservedCertificates := 30 // Número de certificados reservados para equilibrio
	
	// Si tenemos menos de 30 certificados en total, ajustamos
	if estimatedNumCertificates <= reservedCertificates {
		reservedCertificates = estimatedNumCertificates / 3 // Un tercio para equilibrio
		if reservedCertificates < 1 {
			reservedCertificates = 1
		}
	}
	
	// Crear certificados optimizados
	var certificates []Certificate
	certificateID := 1
	
	// Primero agrupamos las órdenes por comerciante para mantener cohesión
	merchantOrders := make(map[int][]Order)
	for _, order := range orders {
		merchantOrders[order.MerchantID] = append(merchantOrders[order.MerchantID], order)
	}

	
	// Cantidad de órdenes a procesar en la primera fase (certificados maxímamente llenos)
	numMainCertificates := estimatedNumCertificates - reservedCertificates
	if numMainCertificates < 1 {
		numMainCertificates = 1
	}
	
	// Implementamos un algoritmo First-Fit-Decreasing para el empaquetado (bin packing)
	// Primero ordenamos las órdenes por monto de mayor a menor
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].Amount > orders[j].Amount
	})
	
	// Estructura para representar un certificado en construcción
	type CertificateBuilder struct {
		Orders []Order
		Amount float64
	}
	
	// Crear los certificados para la primera fase (bin packing)
	certificateBuilders := make([]CertificateBuilder, 0, numMainCertificates)
	
	// Primera fase: Bin Packing con First-Fit-Decreasing
	var remainingOrders []Order
	
	// Procesar las órdenes más grandes primero
	for _, order := range orders {
		// Verificar que esta orden no exceda por sí misma el límite
		if order.Amount > limitAmount {
			fmt.Printf("ADVERTENCIA: Orden ID %d excede el límite por sí misma: $%.2f\n", 
				order.ID, order.Amount)
			// En este caso, podríamos dividir la orden, pero por ahora solo la reportamos
			// y la tratamos como cualquier otra orden
		}
		
		placed := false
		
		// Intentar colocar la orden en un certificado existente
		for i := range certificateBuilders {
			// Verificación ESTRICTA: la suma debe ser EXACTAMENTE menor o igual al límite
			if certificateBuilders[i].Amount + order.Amount <= limitAmount {
				certificateBuilders[i].Orders = append(certificateBuilders[i].Orders, order)
				certificateBuilders[i].Amount += order.Amount
				placed = true
				break
			}
		}
		
		// Si no pudimos colocar la orden en ningún certificado existente
		if !placed {
			// Si tenemos menos certificados que el objetivo, creamos uno nuevo
			if len(certificateBuilders) < numMainCertificates {
				certificateBuilders = append(certificateBuilders, CertificateBuilder{
					Orders: []Order{order},
					Amount: order.Amount,
				})
			} else {
				// Si ya tenemos suficientes certificados principales, 
				// esta orden irá a los certificados de equilibrio
				remainingOrders = append(remainingOrders, order)
			}
		}
	}
	
	// Convertir los constructores de certificados a certificados reales
	for _, builder := range certificateBuilders {
		// Verificación final para asegurar que ningún certificado exceda el límite
		if builder.Amount > limitAmount {
			fmt.Printf("ERROR: Certificado ID %d excede el límite: $%.2f\n", 
				certificateID, builder.Amount)
			// Esto no debería ocurrir dado nuestro algoritmo, pero verificamos por seguridad
		}
		
		certificates = append(certificates, Certificate{
			ID:     certificateID,
			Amount: builder.Amount,
			Orders: append([]Order{}, builder.Orders...),
		})
		certificateID++
	}
	
	// Procesar órdenes restantes para los certificados de equilibrio
	if len(remainingOrders) > 0 {
		// Si no hay órdenes restantes, no hay nada más que hacer
		// Calcular el monto total restante
		remainingAmount := 0.0
		for _, order := range remainingOrders {
			remainingAmount += order.Amount
		}
		
		// Calcular el monto objetivo por certificado de equilibrio
		targetAmountPerBalanceCert := remainingAmount / float64(reservedCertificates)
		if targetAmountPerBalanceCert > limitAmount {
			targetAmountPerBalanceCert = limitAmount * 0.9 // Ajustar para no exceder el límite
		}
		
		// Crear certificados de equilibrio
		currentBalanceCert := CertificateBuilder{}
		balanceCertCount := 0
		
		for _, order := range remainingOrders {
			// PRIMERO verificamos si añadir esta orden excedería el límite absoluto
			if currentBalanceCert.Amount + order.Amount > limitAmount {
				// Finalizar este certificado
				certificates = append(certificates, Certificate{
					ID:     certificateID,
					Amount: currentBalanceCert.Amount,
					Orders: append([]Order{}, currentBalanceCert.Orders...),
				})
				certificateID++
				balanceCertCount++
				
				// Comenzar un nuevo certificado con esta orden
				currentBalanceCert = CertificateBuilder{
					Orders: []Order{order},
					Amount: order.Amount,
				}
				continue // Continuar con la siguiente orden
			}
			
			// Si este certificado ya está cerca del objetivo y añadir esta orden lo sobrepasaría significativamente
			if currentBalanceCert.Amount > 0 && 
			   currentBalanceCert.Amount >= targetAmountPerBalanceCert * 0.85 && 
			   currentBalanceCert.Amount + order.Amount > targetAmountPerBalanceCert * 1.15 &&
			   balanceCertCount < reservedCertificates - 1 {
				// Finalizar este certificado
				certificates = append(certificates, Certificate{
					ID:     certificateID,
					Amount: currentBalanceCert.Amount,
					Orders: append([]Order{}, currentBalanceCert.Orders...),
				})
				certificateID++
				balanceCertCount++
				
				// Comenzar un nuevo certificado con esta orden
				currentBalanceCert = CertificateBuilder{
					Orders: []Order{order},
					Amount: order.Amount,
				}
			} else {
				// Añadir la orden al certificado actual
				currentBalanceCert.Orders = append(currentBalanceCert.Orders, order)
				currentBalanceCert.Amount += order.Amount
			}
		}
		
		// Añadir el último certificado de equilibrio si hay órdenes pendientes
		if len(currentBalanceCert.Orders) > 0 {
			// Verificación final para asegurar que ningún certificado exceda el límite
			if currentBalanceCert.Amount > limitAmount {
				fmt.Printf("ERROR: Último certificado ID %d excede el límite: $%.2f\n", 
					certificateID, currentBalanceCert.Amount)
				// Esto no debería ocurrir dado nuestro algoritmo, pero verificamos por seguridad
			}
			
			certificates = append(certificates, Certificate{
				ID:     certificateID,
				Amount: currentBalanceCert.Amount,
				Orders: append([]Order{}, currentBalanceCert.Orders...),
			})
		}
	}
	
	// Verificación final para todos los certificados
	for _, cert := range certificates {
		if cert.Amount > limitAmount {
			fmt.Printf("ERROR CRÍTICO: Certificado final ID %d excede el límite: $%.2f\n", 
				cert.ID, cert.Amount)
			// Esto es una verificación de seguridad, no debería ocurrir
		}
	}
	
	return certificates
}
	

func main() {
	fmt.Println("Iniciando generación de órdenes...")
	startTime := time.Now()
	
	orders, err := generateOrders()
	if err != nil {
		fmt.Printf("Error al generar órdenes: %v\n", err)
		return
	}
	
	elapsed := time.Since(startTime)
	totalOrders := len(orders)
	fmt.Printf("Se generaron %d órdenes en %v\n", totalOrders, elapsed)
	
	// Mostrar algunas órdenes de ejemplo
	fmt.Println("\nEjemplo de las primeras 5 órdenes:")
	for i := 0; i < 5 && i < len(orders); i++ {
		fmt.Printf("  Orden ID: %d, Comerciante: %d, Monto: $%.2f\n", 
			orders[i].ID, orders[i].MerchantID, orders[i].Amount)
	}
	
	// Calcular el monto total de todas las órdenes
	var totalAmount float64
	for _, order := range orders {
		totalAmount += order.Amount
	}
	
	// Generar certificados con un límite de $500,000 por certificado
	const certificateLimitAmount = 500000.0
	certificates := generateCertificates(orders, certificateLimitAmount)
	
	// Calcular estadísticas de certificados
	var totalCertificateAmount float64
	var minCertAmount float64 = float64(^uint(0) >> 1) // Valor máximo para float64
	var maxCertAmount float64 = 0
	certificateAmounts := make([]float64, len(certificates))
	
	for i, cert := range certificates {
		totalCertificateAmount += cert.Amount
		certificateAmounts[i] = cert.Amount
		
		if cert.Amount < minCertAmount {
			minCertAmount = cert.Amount
		}
		if cert.Amount > maxCertAmount {
			maxCertAmount = cert.Amount
		}
	}
	
	// Calcular el número de certificados teórico basado en la división del monto total
	theoreticalNumCertificates := totalAmount / certificateLimitAmount
	
	// Calcular el porcentaje promedio de llenado de los certificados
	avgFillPercentage := (totalCertificateAmount / float64(len(certificates))) / certificateLimitAmount * 100
	
	// Ordenar los montos para calcular percentiles
	sort.Float64s(certificateAmounts)
	
	// Calcular percentiles relevantes
	p25 := percentile(certificateAmounts, 25)
	p50 := percentile(certificateAmounts, 50) // mediana
	p75 := percentile(certificateAmounts, 75)
	p90 := percentile(certificateAmounts, 90)
	
	// Mostrar estadísticas
	fmt.Println("\nEstadísticas:")
	fmt.Printf("  Número total de comerciantes: 3,500\n")
	fmt.Printf("  Órdenes por comerciante: 612\n")
	fmt.Printf("  Número total de órdenes: %d\n", totalOrders)
	fmt.Printf("  Monto total de órdenes: $%.2f\n", totalAmount)
	fmt.Printf("  Límite por certificado: $%.2f\n", certificateLimitAmount)
	fmt.Printf("  Número teórico de certificados (total/500K): %.2f\n", theoreticalNumCertificates)
	fmt.Printf("  Número real de certificados generados: %d\n", len(certificates))
	fmt.Printf("  Porcentaje promedio de llenado: %.2f%%\n", avgFillPercentage)
	
	fmt.Println("\nDistribución de montos en certificados:")
	fmt.Printf("  Monto mínimo: $%.2f (%.2f%% del límite)\n", minCertAmount, minCertAmount/certificateLimitAmount*100)
	fmt.Printf("  Percentil 25: $%.2f (%.2f%% del límite)\n", p25, p25/certificateLimitAmount*100)
	fmt.Printf("  Mediana (P50): $%.2f (%.2f%% del límite)\n", p50, p50/certificateLimitAmount*100)
	fmt.Printf("  Percentil 75: $%.2f (%.2f%% del límite)\n", p75, p75/certificateLimitAmount*100)
	fmt.Printf("  Percentil 90: $%.2f (%.2f%% del límite)\n", p90, p90/certificateLimitAmount*100)
	fmt.Printf("  Monto máximo: $%.2f (%.2f%% del límite)\n", maxCertAmount, maxCertAmount/certificateLimitAmount*100)
	
	if len(certificates) > 0 {
		// Mostrar ejemplo de certificados (primeros y últimos)
		fmt.Println("\nPrimeros 3 certificados:")
		for i := 0; i < 3 && i < len(certificates); i++ {
			fmt.Printf("  Certificado ID: %d, Monto: $%.2f (%.2f%%), Órdenes: %d\n", 
				certificates[i].ID, certificates[i].Amount, 
				certificates[i].Amount/certificateLimitAmount*100, len(certificates[i].Orders))
		}
		
		fmt.Println("\nÚltimos 3 certificados (de equilibrio):")
		for i := len(certificates) - 3; i < len(certificates); i++ {
			fmt.Printf("  Certificado ID: %d, Monto: $%.2f (%.2f%%), Órdenes: %d\n", 
				certificates[i].ID, certificates[i].Amount,
				certificates[i].Amount/certificateLimitAmount*100, len(certificates[i].Orders))
		}
	}
}

// Función para calcular percentiles
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// Asegurarse de que los valores estén ordenados
	// (asumimos que ya están ordenados si esta función se llama después de sort.Float64s)
	
	// Calcular el índice
	index := float64(len(values)-1) * p / 100
	
	// Si el índice es un entero
	if index == float64(int(index)) {
		return values[int(index)]
	}
	
	// Si es necesario interpolar
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	weight := index - float64(lower)
	
	return values[lower]*(1-weight) + values[upper]*weight
}