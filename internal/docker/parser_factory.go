package docker

import (
	"fmt"
	"strings"
	"sync"
)

// ParserFactory administra las estrategias disponibles para interpretar
// los resultados de ejecución de pruebas en distintos lenguajes.
type ParserFactory struct {
	mu          sync.RWMutex
	parsers     []TestResultParser
	indexByType map[string]int
	fallback    TestResultParser
}

// NewParserFactory crea una nueva instancia de ParserFactory y registra
// las estrategias provistas.
func NewParserFactory(parsers ...TestResultParser) *ParserFactory {
	factory := &ParserFactory{
		indexByType: make(map[string]int),
	}
	if len(parsers) > 0 {
		factory.RegisterParser(parsers...)
	}
	return factory
}

// DefaultParserFactory retorna una instancia lista para usarse con el
// parser de doctest configurado como fallback.
func DefaultParserFactory() *ParserFactory {
	doctest := NewDoctestParser()
	factory := NewParserFactory(doctest)
	factory.SetFallback(doctest)
	return factory
}

// RegisterParser añade nuevas estrategias a la fábrica. En caso de que ya
// exista un parser del mismo tipo, se reemplaza por el nuevo.
func (f *ParserFactory) RegisterParser(parsers ...TestResultParser) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.indexByType == nil {
		f.indexByType = make(map[string]int)
	}

	for _, parser := range parsers {
		if parser == nil {
			continue
		}

		parserType := fmt.Sprintf("%T", parser)
		if idx, exists := f.indexByType[parserType]; exists {
			f.parsers[idx] = parser
			continue
		}

		f.indexByType[parserType] = len(f.parsers)
		f.parsers = append(f.parsers, parser)
	}
}

// SetFallback define el parser a utilizar cuando no existe una estrategia
// específica para el lenguaje solicitado.
func (f *ParserFactory) SetFallback(parser TestResultParser) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fallback = parser
}

// GetParserForLanguage devuelve la estrategia adecuada para el lenguaje
// solicitado. Si no se encuentra una específica, se retorna el fallback.
// Si tampoco existe un fallback, se produce un error.
func (f *ParserFactory) GetParserForLanguage(language string) (TestResultParser, error) {
	lang := strings.TrimSpace(strings.ToLower(language))

	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, parser := range f.parsers {
		if parser.CanParse(lang) {
			return parser, nil
		}
	}

	if f.fallback != nil {
		return f.fallback, nil
	}

	return nil, fmt.Errorf("no parser registered for language %q", language)
}
