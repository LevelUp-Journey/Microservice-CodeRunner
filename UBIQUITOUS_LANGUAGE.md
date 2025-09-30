# 📚 Diccionario del Lenguaje Ubicuo - Microservice CodeRunner

## 🎯 **Entidades Principales del Dominio**

### **Execution (Ejecución)**
> Una instancia completa de evaluación de código que representa la ejecución de una solución de estudiante para un desafío específico.

### **Solution (Solución)**
> El código fuente proporcionado por un estudiante para resolver un desafío de programación.

### **Challenge (Desafío)**
> Un problema de programación definido que debe ser resuelto por los estudiantes, incluyendo especificaciones, restricciones y casos de prueba.

### **Student (Estudiante)**
> Usuario que envía soluciones de código para evaluación en los desafíos.

### **Test Case (Caso de Prueba)**
> Conjunto de datos de entrada y salida esperada utilizados para validar la corrección de una solución.

### **Pipeline (Tubería)**
> Arquitectura modular que orquesta la ejecución de código a través de pasos secuenciales (validación, compilación, obtención de pruebas, ejecución, limpieza).

## 🔄 **Estados y Ciclo de Vida**

### **Execution Status (Estado de Ejecución)**
> Estados posibles: `PENDING` (pendiente), `RUNNING` (ejecutándose), `COMPLETED` (completada), `FAILED` (fallida), `TIMEOUT` (tiempo agotado), `CANCELLED` (cancelada).

### **Step Status (Estado de Paso)**
> Estados de cada paso del pipeline: `PENDING`, `RUNNING`, `COMPLETED`, `FAILED`, `SKIPPED`.

## 🏗️ **Componentes Arquitecturales**

### **Pipeline Step (Paso del Pipeline)**
> Unidad modular ejecutable que representa una fase específica del proceso de evaluación (validación, compilación, etc.).

### **Event Handler (Manejador de Eventos)**
> Componente que procesa eventos del pipeline para logging, monitoreo y persistencia de estado.

### **Logger (Registrador)**
> Sistema de logging estructurado que registra eventos del pipeline con niveles: `DEBUG`, `INFO`, `WARN`, `ERROR`.

## 💾 **Persistencia y Datos**

### **Execution Repository (Repositorio de Ejecuciones)**
> Capa de acceso a datos que maneja operaciones CRUD para registros de ejecuciones.

### **Execution Metadata (Metadatos de Ejecución)**
> Información adicional sobre la ejecución incluyendo tiempos, memoria utilizada, códigos de salida y resultados de compilación.

### **Compilation Result (Resultado de Compilación)**
> Información sobre el proceso de compilación incluyendo éxito/fallo, mensajes de error y advertencias.

## ⚙️ **Configuración y Recursos**

### **Execution Config (Configuración de Ejecución)**
> Parámetros que controlan la ejecución: límites de tiempo, memoria, variables de entorno, modo debug.

### **Working Directory (Directorio de Trabajo)**
> Espacio temporal donde se ejecuta el código, incluyendo archivos fuente, compilados y de salida.

### **Resource Limits (Límites de Recursos)**
> Restricciones aplicadas durante la ejecución: tiempo máximo, memoria máxima, acceso a red.

## 🔧 **Pasos del Pipeline**

### **Validation Step (Paso de Validación)**
> Primer paso que valida entrada, lenguaje soportado, configuración y contenido del código.

### **Compilation Step (Paso de Compilación)**
> Compila código fuente para lenguajes compilados (Java, C++, Go, Rust, C#).

### **Test Fetching Step (Paso de Obtención de Pruebas)**
> Recupera casos de prueba asociados al desafío desde sistemas externos.

### **Execution Step (Paso de Ejecución)**
> Ejecuta el código compilado/interpreted contra todos los casos de prueba.

### **Cleanup Step (Paso de Limpieza)**
> Elimina archivos temporales, limpia recursos del sistema y remueve datos sensibles.

## 🌐 **Comunicación y APIs**

### **gRPC Service (Servicio gRPC)**
> API de alto rendimiento basada en gRPC para comunicación entre microservicios.

### **Execution Request (Solicitud de Ejecución)**
> Mensaje gRPC que inicia una nueva ejecución con solución, desafío, estudiante y configuración.

### **Execution Response (Respuesta de Ejecución)**
> Resultado de la ejecución incluyendo estado, casos de prueba aprobados y metadatos.

### **Stream Logs (Flujo de Logs)**
> Streaming en tiempo real de logs de ejecución para monitoreo y debugging.

## 📊 **Métricas y Monitoreo**

### **Execution Time (Tiempo de Ejecución)**
> Duración total del proceso de evaluación en milisegundos.

### **Memory Usage (Uso de Memoria)**
> Cantidad de memoria utilizada durante la ejecución en MB.

### **Test Coverage (Cobertura de Pruebas)**
> Porcentaje de casos de prueba que pasan exitosamente.

### **Approved Test IDs (IDs de Pruebas Aprobadas)**
> Lista de identificadores de casos de prueba que la solución pasó correctamente.

## 🔒 **Seguridad y Aislamiento**

### **Sandboxing (Aislamiento)**
> Entorno controlado que limita el acceso del código ejecutado al sistema host.

### **Resource Isolation (Aislamiento de Recursos)**
> Separación de recursos entre ejecuciones para prevenir interferencias.

### **Sensitive Data Sanitization (Saneamiento de Datos Sensibles)**
> Eliminación de información sensible de logs y resultados antes de almacenamiento.

## 🗃️ **Base de Datos**

### **Execution Record (Registro de Ejecución)**
> Entidad persistida que almacena el historial completo de cada evaluación.

### **Execution Step (Paso de Ejecución en BD)**
> Registro detallado de cada paso del pipeline con timestamps y estado.

### **Execution Log (Log de Ejecución)**
> Entradas de log persistidas para auditoría y debugging.

### **Test Result (Resultado de Prueba)**
> Registro individual de cada caso de prueba ejecutado con resultado y métricas.

## 🐳 **Infraestructura**

### **Containerization (Contenedorización)**
> Empaquetamiento del servicio en contenedores Docker para portabilidad.

### **Database Connection Pool (Pool de Conexiones)**
> Gestión eficiente de conexiones a PostgreSQL con métricas de uso.

### **Health Check (Verificación de Salud)**
> Endpoint para monitoreo de la disponibilidad del servicio.

---

## 🎯 **Términos por Importancia**

### **Críticos para el Dominio:**
1. **Execution** - Concepto central
2. **Solution** - Lo que se evalúa
3. **Challenge** - El problema a resolver
4. **Pipeline** - Arquitectura de ejecución
5. **Test Case** - Validación de corrección

### **Importantes para Operaciones:**
6. **Execution Status** - Estado del proceso
7. **Pipeline Step** - Componentes modulares
8. **Execution Config** - Parámetros de control
9. **Resource Limits** - Restricciones de seguridad
10. **Execution Metadata** - Información de resultados

### **Técnicos pero Relevantes:**
11. **gRPC Service** - Interfaz de comunicación
12. **Working Directory** - Entorno de ejecución
13. **Event Handler** - Procesamiento de eventos
14. **Logger** - Sistema de observabilidad
15. **Execution Repository** - Persistencia de datos

---

## 📋 **Resumen Ejecutivo**

Este **Microservice CodeRunner** implementa un **lenguaje ubicuo** para el dominio de **evaluación automatizada de código** en plataformas educativas. El dominio se centra en:

- **Ejecución segura y aislada** de código estudiante
- **Validación automática** contra casos de prueba
- **Arquitectura pipeline modular** para extensibilidad
- **Monitoreo completo** y persistencia de resultados
- **Integración gRPC** para comunicación de alto rendimiento

**Características clave del dominio:**
- 🔒 **Seguridad primero**: Aislamiento, límites de recursos, saneamiento de datos
- 📊 **Observabilidad completa**: Logging estructurado, métricas, eventos
- 🏗️ **Arquitectura modular**: Pipeline extensible con pasos intercambiables
- ⚡ **Alto rendimiento**: gRPC, ejecución concurrente, optimización de recursos
- 🎯 **Enfoque educativo**: Evaluación objetiva, feedback detallado, escalabilidad