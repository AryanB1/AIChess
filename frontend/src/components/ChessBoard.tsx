import React, { useState } from 'react';
import { Chessboard } from 'react-chessboard';
import { Chess } from 'chess.js';

function ChessBoard() {
    // Initializes the chess engine
    const engine = new Chess();
    // Defines the board orientation types
    type BoardOrientation = 'white' | 'black';
    // State to manage the current board position in FEN format
    const [boardState, setBoardState] = useState(engine.fen());
    // State to manage the board's orientation
    const [boardOrientationState, setBoardOrientationState] = useState<BoardOrientation>('white');
    // State to manage whether the board is locked
    const [isLocked, setIsLocked] = useState(false);
    // State to track if the game is over
    const [gameState, setGameState] = useState(false);
    // Function to flip the board orientation
    const flipBoard = () => {
        setBoardOrientationState((prev) => (prev === 'white' ? 'black' : 'white'));
    };
    // Function to play a random move on the board
    const playRandomMove = () => {
        // If the board is locked, don't allow a move
        if (isLocked) return;
        // Loads the current board state into the chess engine
        engine.load(boardState)
        // Gets all legal moves
        const legalMoves = engine.moves({verbose:true});
        // Chooses a random move and keep retrying until we have a valid move
        let randomIndex = Math.floor(Math.random() * legalMoves.length);
        while (true) {
            let validator = onDrop(legalMoves[randomIndex].from, legalMoves[randomIndex].to);
            if (validator) break;
            randomIndex = Math.floor(Math.random() * legalMoves.length);
        }
        // Checks if game is over
        if(engine.isGameOver()) {
            setIsLocked(true)
            setGameState(true)
            return true;
        }
    };
    const onDrop = (sourceSquare: string, targetSquare: string): boolean => {
        // If the board is locked, don't allow a move
        if (isLocked) return false;
        // Loads the current board state into the chess engine
        engine.load(boardState)
        try {
            // Checks if move is valid
            const move = engine.move({
                from: sourceSquare,
                to: targetSquare,
            });
            // If it is valid, update board state, otherwise throw error
            if (move) {
                setBoardState(engine.fen());
                setIsLocked(true);
                // checks if game is finished
                if(engine.isGameOver()) {
                    setIsLocked(true)
                    setGameState(true)
                    return true;
                }
                stockFishMove(engine.fen()).then(() => {
                    // Unlock the board after handling the API response
                    setIsLocked(false);
                }).catch(error => {
                    console.error('Error in API:', error);
                    // Unlock the board if API error
                    setIsLocked(false);
                });
                // Checks if game is over
                if(engine.isGameOver()){
                    console.log("game won!")
                    setGameState(true)
                    setIsLocked(true)
                }
                return true;
            }
        } catch (error) {
            console.error("Error making move:", error);
        }
        return false;
    };
    // Calls stockfish API and moves a chess piece
    const stockFishMove = async (board: string): Promise<void> => {
        try {
            // Sends the request and throws error if request fails
            const response = await fetch("http://localhost:8081/engine-move", {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    depth: 12,
                    fen: board,
                }),
            });

            if (!response.ok) {
                throw new Error('Network response was not ok');
            }

            const data = await response.json();
            console.log('API response:', data);
            // Extracts values from API response
            const bestMove = data.bestmove;
            const success = data.success;
            // Completes the move, updates fen, and checks if game is over
            if (bestMove && success) {
                engine.load(board)
                engine.move(bestMove)
                setBoardState(engine.fen())
                if(engine.isGameOver()) {
                    setIsLocked(true)
                    setGameState(true)
                }
            }
            else {
                throw new Error('API response was not ok');
            }

        } catch (error) {
            console.error('Error calling the API:', error);
        }
    };
    // Resets game
    const ResetGame = () => {
        engine.reset()
        setBoardState(engine.fen())
        setIsLocked(false)
        setGameState(false)
    }
    return (
        <div style={{ backgroundColor: '#f4f4f9', minHeight: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '20px' }}>
            <div style={{ marginBottom: '20px' }}>
                {gameState && <div style={{ fontSize: '18px', fontWeight: 'bold', color: 'red' }}>Game is over!</div>}
            </div>
            <div style={{ width: '500px', marginBottom: '20px' }}>
                <Chessboard
                    id="BasicBoard"
                    boardOrientation={boardOrientationState}
                    showBoardNotation={true}
                    position={boardState}
                    onPieceDrop={onDrop}
                />
            </div>
            <div style={{ display: 'flex', gap: '10px' }}>
                <button
                    onClick={playRandomMove}
                    disabled={isLocked}
                    style={{ padding: '10px 20px', backgroundColor: '#007bff', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}
                >
                    Play Random Move
                </button>
                <button
                    onClick={flipBoard}
                    style={{ padding: '10px 20px', backgroundColor: '#28a745', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}
                >
                    Flip Board
                </button>
                <button
                    onClick={ResetGame}
                    style={{ padding: '10px 20px', backgroundColor: '#dc3545', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}
                >
                    Reset Game
                </button>
            </div>
        </div>
    );
}

export default ChessBoard;
