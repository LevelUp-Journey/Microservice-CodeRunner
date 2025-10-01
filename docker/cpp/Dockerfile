# Dockerfile para ejecutar código C++ con doctest
FROM gcc:13.2

# Instalar herramientas básicas
RUN apt-get update && apt-get install -y \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Crear directorio de trabajo
WORKDIR /workspace

# Descargar e instalar doctest (header-only library)
RUN wget https://github.com/doctest/doctest/releases/download/v2.4.11/doctest.h -O /usr/local/include/doctest.h

# Crear directorio para temporales (necesario para compilación)
RUN mkdir -p /tmp/workspace && chmod 777 /tmp/workspace

# Configurar variables de entorno
ENV LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    DEBIAN_FRONTEND=noninteractive

# El código se montará en /workspace
# El contenedor ejecutará: g++ -std=c++17 solution.cpp -o solution && ./solution

# Usuario no root para seguridad
RUN useradd -m -u 1000 coderunner && \
    chown -R coderunner:coderunner /workspace /tmp/workspace

USER coderunner

CMD ["/bin/bash"]
