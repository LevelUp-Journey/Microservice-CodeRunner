# Java testing environment with JUnit
FROM openjdk:17-jdk-slim-bullseye

LABEL maintainer="LevelUp Journey"
LABEL description="Java execution environment with JUnit framework"

# Install system dependencies
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    maven \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /workspace

# Create Maven project structure
RUN mkdir -p src/main/java src/test/java

# Create pom.xml with JUnit dependencies
RUN echo '<?xml version="1.0" encoding="UTF-8"?>\
<project xmlns="http://maven.apache.org/POM/4.0.0"\
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"\
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0\
         http://maven.apache.org/xsd/maven-4.0.0.xsd">\
    <modelVersion>4.0.0</modelVersion>\
    <groupId>com.levelup</groupId>\
    <artifactId>code-runner-test</artifactId>\
    <version>1.0.0</version>\
    <properties>\
        <maven.compiler.source>17</maven.compiler.source>\
        <maven.compiler.target>17</maven.compiler.target>\
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>\
    </properties>\
    <dependencies>\
        <dependency>\
            <groupId>org.junit.jupiter</groupId>\
            <artifactId>junit-jupiter</artifactId>\
            <version>5.10.0</version>\
            <scope>test</scope>\
        </dependency>\
    </dependencies>\
    <build>\
        <plugins>\
            <plugin>\
                <groupId>org.apache.maven.plugins</groupId>\
                <artifactId>maven-surefire-plugin</artifactId>\
                <version>3.1.2</version>\
                <configuration>\
                    <includes>\
                        <include>**/*Test.java</include>\
                    </includes>\
                </configuration>\
            </plugin>\
        </plugins>\
    </build>\
</project>' > pom.xml

# Download dependencies
RUN mvn dependency:resolve

# Set JVM memory limits
ENV JAVA_OPTS="-Xmx512m -Xms128m"

# Default command
CMD ["/bin/bash"]